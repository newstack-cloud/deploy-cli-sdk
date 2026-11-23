package stateio

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

// An in-memory object store that counts requests, standing in for a bucket to measure how
// many an import or export makes.
type countingObjectStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	reads   map[string]int
	writes  map[string]int
	lists   int
}

func newCountingObjectStore() *countingObjectStore {
	return &countingObjectStore{objects: map[string][]byte{}, reads: map[string]int{}, writes: map[string]int{}}
}

func (s *countingObjectStore) Get(
	_ context.Context,
	key string,
) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads[key] += 1
	data, ok := s.objects[key]
	if !ok {
		return nil, "", objectstore.NewObjectNotFound(key)
	}
	return data, etagFor(data), nil
}

func (s *countingObjectStore) Put(
	_ context.Context,
	key string,
	data []byte,
	opts *objectstore.PutOptions,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.objects[key]
	if opts != nil && opts.IfNoneMatch == "*" && exists {
		return "", objectstore.NewPreconditionFailed(key)
	}

	if opts != nil && opts.IfMatch != "" &&
		(!exists || etagFor(existing) != opts.IfMatch) {
		return "", objectstore.NewPreconditionFailed(key)
	}

	s.objects[key] = data
	s.writes[key] += 1
	return etagFor(data), nil
}

func (s *countingObjectStore) Delete(_ context.Context, key string, opts *objectstore.DeleteOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, exists := s.objects[key]
	if opts != nil && opts.IfMatch != "" &&
		(!exists || etagFor(existing) != opts.IfMatch) {
		return objectstore.NewPreconditionFailed(key)
	}
	delete(s.objects, key)
	return nil
}

func (s *countingObjectStore) Head(_ context.Context, key string) (*objectstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, objectstore.NewObjectNotFound(key)
	}
	return &objectstore.ObjectInfo{Key: key, Size: int64(len(data)), ETag: etagFor(data)}, nil
}

func (s *countingObjectStore) List(_ context.Context, prefix string) ([]*objectstore.ObjectInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lists += 1
	objects := []*objectstore.ObjectInfo{}
	for key, data := range s.objects {
		if strings.HasPrefix(key, prefix) {
			objects = append(
				objects,
				&objectstore.ObjectInfo{
					Key:  key,
					Size: int64(len(data)),
					ETag: etagFor(data),
				},
			)
		}
	}
	return objects, nil
}

func (s *countingObjectStore) resetCounts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads = map[string]int{}
	s.writes = map[string]int{}
	s.lists = 0
}

func etagFor(data []byte) string {
	return fmt.Sprintf("%08x", crc32.ChecksumIEEE(data))
}

type ObjectStoreExportRequestsTestSuite struct {
	suite.Suite
	store *countingObjectStore
}

func (s *ObjectStoreExportRequestsTestSuite) SetupTest() {
	s.store = newCountingObjectStore()
	seed := s.freshContainer()

	instances := []state.InstanceState{}
	for i := range 20 {
		child := &state.InstanceState{
			InstanceID:   fmt.Sprintf("child-%d", i),
			InstanceName: fmt.Sprintf("child-name-%d", i),
			Status:       core.InstanceStatusDeployed,
		}
		instances = append(instances, state.InstanceState{
			InstanceID:      fmt.Sprintf("parent-%d", i),
			InstanceName:    fmt.Sprintf("parent-name-%d", i),
			Status:          core.InstanceStatusDeployed,
			ChildBlueprints: map[string]*state.InstanceState{"child": child},
		})
	}
	s.Require().NoError(NewContainerStateImporter(seed).ImportInstances(context.Background(), instances))
	s.store.resetCounts()
}

// Each export runs in a fresh process, so it starts from an empty cache like a CI run.
func (s *ObjectStoreExportRequestsTestSuite) freshContainer() state.Container {
	container, err := objectstore.LoadStateContainer(context.Background(), s.store, "state", core.NewNopLogger())
	s.Require().NoError(err)
	return container
}

func (s *ObjectStoreExportRequestsTestSuite) Test_export_all_reads_each_instance_object_once() {
	exported, err := NewContainerStateExporter(s.freshContainer()).
		ExportInstances(context.Background(), nil)

	s.Require().NoError(err)
	// Each child is exported only under its parent.
	s.Len(exported, 20)
	for _, parent := range exported {
		s.Contains(parent.ChildBlueprints, "child", parent.InstanceID)
	}
	s.Equal(1, s.store.lists)
	s.Len(s.store.reads, 40)
	for key, reads := range s.store.reads {
		s.Equal(1, reads, key)
	}
}

func (s *ObjectStoreExportRequestsTestSuite) Test_filtered_export_reads_only_the_requested_subtrees() {
	exported, err := NewContainerStateExporter(s.freshContainer()).ExportInstances(
		context.Background(),
		[]string{"parent-name-1", "parent-2"},
	)

	s.Require().NoError(err)
	s.Len(exported, 2)
	s.Equal(0, s.store.lists)
	for key, reads := range s.store.reads {
		s.Equal(1, reads, key)
	}
	// Two parents and their children, the name record behind the name filter, and the
	// lookup of that name as an ID, which misses, since a filter can be either.
	s.Len(s.store.reads, 6)
}

func (s *ObjectStoreExportRequestsTestSuite) Test_import_writes_each_instance_object_once() {
	deployed := core.InstanceStatusDeployed
	grandchild := &state.InstanceState{InstanceID: "grandchild", InstanceName: "grandchild-name", Status: deployed}
	child := &state.InstanceState{
		InstanceID:      "child",
		InstanceName:    "child-name",
		Status:          deployed,
		ChildBlueprints: map[string]*state.InstanceState{"grandchild": grandchild},
	}
	parent := state.InstanceState{
		InstanceID:      "parent",
		InstanceName:    "parent-name",
		Status:          deployed,
		ChildBlueprints: map[string]*state.InstanceState{"child": child},
	}

	// A file can list a child at the top level as well as nested under its parent.
	imported := []state.InstanceState{parent, *child, *grandchild}
	for range 2 {
		s.store.resetCounts()
		err := NewContainerStateImporter(s.freshContainer()).
			ImportInstances(context.Background(), imported)
		s.Require().NoError(err)

		for _, id := range []string{"parent", "child", "grandchild"} {
			s.Equal(1, s.store.writes["state/instances/"+id+".json"], id)
		}
	}
}

func TestObjectStoreExportRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreExportRequestsTestSuite))
}
