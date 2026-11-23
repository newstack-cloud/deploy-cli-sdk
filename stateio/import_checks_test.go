package stateio

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type ImportChecksTestSuite struct {
	suite.Suite
	fs           afero.Fs
	stateDir     string
	engineConfig *EngineConfig
}

func (s *ImportChecksTestSuite) SetupTest() {
	s.fs = afero.NewMemMapFs()
	s.stateDir = "/test/state"
	s.Require().NoError(s.fs.MkdirAll(s.stateDir, 0700))
	s.engineConfig = &EngineConfig{
		State: StateConfig{StorageEngine: StorageEngineMemfile, MemFileStateDir: s.stateDir},
	}
}

func (s *ImportChecksTestSuite) importInstances(force bool, instances ...state.InstanceState) error {
	data, err := json.Marshal(instances)
	s.Require().NoError(err)

	_, err = Import(ImportParams{
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     data,
		Force:        force,
	})
	return err
}

func (s *ImportChecksTestSuite) importJSON(input string) error {
	_, err := Import(ImportParams{
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData:     []byte(input),
	})
	return err
}

func (s *ImportChecksTestSuite) storedInstance(id string) (state.InstanceState, error) {
	container, err := memfile.LoadStateContainer(s.stateDir, s.fs, core.NewNopLogger())
	s.Require().NoError(err)
	return container.Instances().Get(context.Background(), id)
}

func (s *ImportChecksTestSuite) assertImportError(err error, expected ImportErrorCode) *ImportError {
	s.Require().Error(err)
	var importErr *ImportError
	s.Require().True(errors.As(err, &importErr), "expected an ImportError, got %v", err)
	s.Equal(expected, importErr.Code)
	return importErr
}

func instance(id string, name string, status core.InstanceStatus) state.InstanceState {
	return state.InstanceState{InstanceID: id, InstanceName: name, Status: status}
}

func (s *ImportChecksTestSuite) Test_rejects_children_in_a_cycle_and_writes_nothing() {
	_, err := Import(ImportParams{
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData: []byte(`[{
			"id": "a", "name": "a", "status": 2,
			"childBlueprints": {"b": {
				"id": "b", "name": "b", "status": 2,
				"childBlueprints": {"a": {"id": "a", "name": "a", "status": 2}}
			}}
		}]`),
	})

	importErr := s.assertImportError(err, ErrCodeChildBlueprintCycle)
	s.Equal(
		"nothing was imported, these instances are children of each other in a cycle: a -> b -> a",
		importErr.Message,
	)
	_, err = s.storedInstance("a")
	s.True(state.IsInstanceNotFound(err), "expected nothing to be written, got %v", err)
}

func (s *ImportChecksTestSuite) Test_rejects_a_cycle_below_a_top_level_instance_and_writes_nothing() {
	cases := map[string]struct {
		input string
		cycle string
	}{
		"between two children": {
			input: `[{"id": "r", "name": "r", "status": 2, "childBlueprints": {"a": {
				"id": "a", "name": "a", "status": 2, "childBlueprints": {"b": {
					"id": "b", "name": "b", "status": 2, "childBlueprints": {"a": {"id": "a"}}
				}}
			}}}]`,
			cycle: "a -> b -> a",
		},
		"a child of itself": {
			input: `[{"id": "r", "name": "r", "status": 2, "childBlueprints": {"a": {
				"id": "a", "name": "a", "status": 2, "childBlueprints": {"again": {"id": "a"}}
			}}}]`,
			cycle: "a -> a",
		},
	}
	for description, testCase := range cases {
		err := s.importJSON(testCase.input)

		importErr := s.assertImportError(err, ErrCodeChildBlueprintCycle)
		s.Equal(
			"nothing was imported, these instances are children of each other in a cycle: "+testCase.cycle,
			importErr.Message,
			description,
		)
		_, err = s.storedInstance("r")
		s.True(state.IsInstanceNotFound(err), "expected nothing to be written for %s, got %v", description, err)
	}
}

func (s *ImportChecksTestSuite) Test_rejects_an_empty_child_blueprint_entry() {
	err := s.importJSON(`[{"id": "r", "name": "r", "status": 2, "childBlueprints": {"networking": null}}]`)

	importErr := s.assertImportError(err, ErrCodeInvalidJSON)
	s.Equal(`nothing was imported, child blueprint "networking" of instance r is empty`, importErr.Message)
}

func (s *ImportChecksTestSuite) Test_rejects_an_empty_child_blueprint_entry_in_a_repeated_copy() {
	err := s.importJSON(`[
		{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}},
		{"id": "c", "name": "c", "status": 2, "childBlueprints": {"storage": null}}
	]`)

	importErr := s.assertImportError(err, ErrCodeInvalidJSON)
	s.Equal(`nothing was imported, child blueprint "storage" of instance c is empty`, importErr.Message)
}

func (s *ImportChecksTestSuite) Test_rejects_an_instance_without_an_id() {
	cases := map[string]struct {
		input   string
		message string
	}{
		"at the top level": {
			input:   `[{"name": "orders", "status": 2}]`,
			message: "nothing was imported, an instance in the input file has no id",
		},
		"as a child": {
			input:   `[{"id": "r", "name": "r", "status": 2, "childBlueprints": {"networking": {"name": "n"}}}]`,
			message: "nothing was imported, a child blueprint of instance r has no id",
		},
	}
	for description, testCase := range cases {
		err := s.importJSON(testCase.input)

		importErr := s.assertImportError(err, ErrCodeInvalidJSON)
		s.Equal(testCase.message, importErr.Message, description)
	}
}

func (s *ImportChecksTestSuite) Test_rejects_copies_of_an_instance_that_differ_and_writes_nothing() {
	cases := map[string]struct {
		input     string
		differing string
	}{
		"the nested copy first": {
			input: `[
				{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}},
				{"id": "c", "name": "c", "status": 13}
			]`,
			differing: "c",
		},
		"the top-level copy first": {
			input: `[
				{"id": "c", "name": "c", "status": 13},
				{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}}
			]`,
			differing: "c",
		},
		"a child name pointing at another instance": {
			input: `[
				{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}},
				{"id": "q", "name": "q", "status": 2, "childBlueprints": {"p": {
					"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {"id": "d", "name": "d", "status": 2}}
				}}}
			]`,
			differing: "p",
		},
		"below a repeated copy that matches": {
			input: `[
				{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {
					"id": "c", "name": "c", "status": 2, "childBlueprints": {"g": {"id": "g", "name": "g", "status": 2}}
				}}},
				{"id": "c", "name": "c", "status": 2, "childBlueprints": {"g": {"id": "g", "name": "g", "status": 13}}}
			]`,
			differing: "g",
		},
	}
	for description, testCase := range cases {
		err := s.importJSON(testCase.input)

		importErr := s.assertImportError(err, ErrCodeInstanceCopiesDiffer)
		s.Equal(
			"nothing was imported, instance "+testCase.differing+
				" appears more than once in the input file and the copies differ",
			importErr.Message,
			description,
		)
		s.Equal("Export the state again, or make every copy of the instance the same.", importErr.Hint, description)
		for _, id := range []string{"p", "q", "c", "d", "g"} {
			_, err = s.storedInstance(id)
			s.True(state.IsInstanceNotFound(err), "expected %s not to be written for %s, got %v", id, description, err)
		}
	}
}

func (s *ImportChecksTestSuite) Test_rejects_a_child_listed_in_more_than_one_place_and_writes_nothing() {
	cases := map[string]struct {
		input   string
		message string
	}{
		"under two parents": {
			input: `[
				{"id": "p1", "name": "p1", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}},
				{"id": "p2", "name": "p2", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}}
			]`,
			message: `nothing was imported, instance c is a child blueprint in more than one place: ` +
				`"c" of instance p1 and "c" of instance p2`,
		},
		"under two names in one parent": {
			input: `[{"id": "p1", "name": "p1", "status": 2, "childBlueprints": {
				"a": {"id": "c", "name": "c", "status": 2},
				"b": {"id": "c", "name": "c", "status": 2}
			}}]`,
			message: `nothing was imported, instance c is a child blueprint in more than one place: ` +
				`"a" of instance p1 and "b" of instance p1`,
		},
		"a parent nested under two others": {
			input: `[
				{"id": "p1", "name": "p1", "status": 2, "childBlueprints": {"x": {
					"id": "p2", "name": "p2", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}
				}}},
				{"id": "g", "name": "g", "status": 2, "childBlueprints": {"y": {
					"id": "p2", "name": "p2", "status": 2, "childBlueprints": {"c": {"id": "c", "name": "c", "status": 2}}
				}}}
			]`,
			message: `nothing was imported, instance p2 is a child blueprint in more than one place: ` +
				`"x" of instance p1 and "y" of instance g`,
		},
	}
	for description, testCase := range cases {
		err := s.importJSON(testCase.input)

		importErr := s.assertImportError(err, ErrCodeChildBlueprintMultipleParents)
		s.Equal(testCase.message, importErr.Message, description)
		s.Equal("List the instance under only one parent, once.", importErr.Hint, description)
		for _, id := range []string{"p1", "p2", "g", "c"} {
			_, err = s.storedInstance(id)
			s.True(state.IsInstanceNotFound(err), "expected %s not to be written for %s, got %v", id, description, err)
		}
	}
}

func (s *ImportChecksTestSuite) Test_imports_copies_of_an_instance_that_match() {
	result, err := Import(ImportParams{
		EngineConfig: s.engineConfig,
		FileSystem:   s.fs,
		FileData: []byte(`[
			{"id": "p", "name": "p", "status": 2, "childBlueprints": {"c": {
				"id": "c", "name": "c", "status": 2, "childBlueprints": {"g": {"id": "g", "name": "g", "status": 2}}
			}}},
			{"id": "c", "name": "c", "status": 2, "childBlueprints": {"g": {"id": "g", "name": "g", "status": 2}}},
			{"id": "g", "name": "g", "status": 2}
		]`),
	})
	s.Require().NoError(err)
	s.Equal(3, result.InstancesCount)
	for _, id := range []string{"c", "g"} {
		_, err = s.storedInstance(id)
		s.NoError(err, id)
	}

	storedParent, err := s.storedInstance("p")
	s.Require().NoError(err)
	s.Require().Contains(storedParent.ChildBlueprints, "c")
	s.Contains(storedParent.ChildBlueprints["c"].ChildBlueprints, "g")
}

func (s *ImportChecksTestSuite) Test_rejects_a_new_instance_using_an_existing_name_and_writes_nothing() {
	s.Require().NoError(s.importInstances(false, instance("inst-1", "orders", core.InstanceStatusDeployed)))

	err := s.importInstances(
		false,
		instance("inst-2", "billing", core.InstanceStatusDeployed),
		instance("inst-3", "orders", core.InstanceStatusDeployed),
	)

	importErr := s.assertImportError(err, ErrCodeInstanceNameConflict)
	s.Equal(
		"nothing was imported\ninstance names already in use:\n"+
			`  - "orders" is already used by instance inst-1, not inst-3`,
		importErr.Message,
	)
	_, err = s.storedInstance("inst-2")
	s.True(state.IsInstanceNotFound(err), "expected nothing from the failed import to be written")
}

func (s *ImportChecksTestSuite) Test_rejects_two_new_instances_sharing_a_name_in_the_input() {
	err := s.importInstances(
		false,
		instance("inst-1", "orders", core.InstanceStatusDeployed),
		instance("inst-2", "orders", core.InstanceStatusDeployed),
	)

	importErr := s.assertImportError(err, ErrCodeInstanceNameConflict)
	s.Equal(
		"nothing was imported\ninstance names already in use:\n"+
			`  - "orders" is used by both inst-1 and inst-2 in the input file`,
		importErr.Message,
	)
}

func (s *ImportChecksTestSuite) Test_refuses_to_import_over_an_instance_with_an_operation_in_progress() {
	s.Require().NoError(s.importInstances(false, instance("inst-1", "orders", core.InstanceStatusDeploying)))

	err := s.importInstances(false, instance("inst-1", "orders", core.InstanceStatusDeployed))

	importErr := s.assertImportError(err, ErrCodeInstanceOperationInProgress)
	s.Equal(
		"nothing was imported\ninstances with an operation in progress:\n"+
			`  - inst-1 ("orders") is DEPLOYING`,
		importErr.Message,
	)
	s.Equal(
		"If an operation was abandoned without its status being updated, "+
			"import again with --force to overwrite the instance anyway.",
		importErr.Hint,
	)
	stored, err := s.storedInstance("inst-1")
	s.Require().NoError(err)
	s.Equal(core.InstanceStatusDeploying, stored.Status)
}

func (s *ImportChecksTestSuite) Test_force_imports_over_an_instance_with_an_operation_in_progress() {
	s.Require().NoError(s.importInstances(false, instance("inst-1", "orders", core.InstanceStatusDestroying)))

	err := s.importInstances(true, instance("inst-1", "orders", core.InstanceStatusDeployed))

	s.Require().NoError(err)
	stored, err := s.storedInstance("inst-1")
	s.Require().NoError(err)
	s.Equal(core.InstanceStatusDeployed, stored.Status)
}

func (s *ImportChecksTestSuite) Test_imports_over_settled_statuses_without_force() {
	settled := []core.InstanceStatus{
		core.InstanceStatusDeployInterrupted,
		core.InstanceStatusUpdateFailed,
		core.InstanceStatusDestroyRollbackComplete,
	}
	for _, status := range settled {
		s.SetupTest()
		s.Require().NoError(s.importInstances(false, instance("inst-1", "orders", status)))

		err := s.importInstances(false, instance("inst-1", "orders", core.InstanceStatusDeployed))

		s.Require().NoError(err, status.String())
	}
}

func (s *ImportChecksTestSuite) Test_moves_the_version_past_the_stored_and_imported_versions() {
	existing := instance("inst-1", "orders", core.InstanceStatusDeployed)
	existing.Version = 3
	s.Require().NoError(s.importInstances(false, existing))

	older := instance("inst-1", "orders", core.InstanceStatusDeployed)
	older.Version = 1
	s.Require().NoError(s.importInstances(false, older))
	stored, err := s.storedInstance("inst-1")
	s.Require().NoError(err)
	s.Equal(int64(4), stored.Version)

	newer := instance("inst-1", "orders", core.InstanceStatusDeployed)
	newer.Version = 7
	s.Require().NoError(s.importInstances(false, newer))
	stored, err = s.storedInstance("inst-1")
	s.Require().NoError(err)
	s.Equal(int64(8), stored.Version)
}

func (s *ImportChecksTestSuite) Test_moves_the_version_of_an_existing_child_past_the_stored_version() {
	storedChild := instance("child-1", "orders-networking", core.InstanceStatusDeployed)
	storedChild.Version = 3
	storedParent := instance("parent-1", "orders", core.InstanceStatusDeployed)
	storedParent.ChildBlueprints = map[string]*state.InstanceState{"networking": &storedChild}
	s.Require().NoError(s.importInstances(false, storedParent))

	child := instance("child-1", "orders-networking", core.InstanceStatusDeployed)
	child.Version = 1
	parent := instance("parent-1", "orders", core.InstanceStatusDeployed)
	parent.ChildBlueprints = map[string]*state.InstanceState{"networking": &child}
	s.Require().NoError(s.importInstances(false, parent))

	stored, err := s.storedInstance("child-1")
	s.Require().NoError(err)
	s.Equal(int64(4), stored.Version)
}

func (s *ImportChecksTestSuite) Test_checks_children_below_the_first_level() {
	s.Require().NoError(s.importInstances(false, nestedChain(core.InstanceStatusUpdating)))

	err := s.importInstances(false, nestedChain(core.InstanceStatusDeployed))

	importErr := s.assertImportError(err, ErrCodeInstanceOperationInProgress)
	s.Equal(
		"nothing was imported\ninstances with an operation in progress:\n"+
			`  - level-3 ("level-3") is UPDATING`,
		importErr.Message,
	)
}

// A top-level instance with children three levels down, the deepest of which has the
// given status.
func nestedChain(deepestStatus core.InstanceStatus) state.InstanceState {
	deepest := instance("level-3", "level-3", deepestStatus)
	middle := instance("level-2", "level-2", core.InstanceStatusDeployed)
	middle.ChildBlueprints = map[string]*state.InstanceState{"deepest": &deepest}
	upper := instance("level-1", "level-1", core.InstanceStatusDeployed)
	upper.ChildBlueprints = map[string]*state.InstanceState{"middle": &middle}
	top := instance("top", "top", core.InstanceStatusDeployed)
	top.ChildBlueprints = map[string]*state.InstanceState{"upper": &upper}
	return top
}

func (s *ImportChecksTestSuite) Test_updates_nested_children_of_an_existing_parent() {
	child := instance("child-1", "orders-networking", core.InstanceStatusDeployed)
	parent := instance("parent-1", "orders", core.InstanceStatusDeployed)
	parent.ChildBlueprints = map[string]*state.InstanceState{"networking": &child}
	s.Require().NoError(s.importInstances(false, parent))

	// A filtered export lists children only nested under their parent.
	updatedChild := instance("child-1", "orders-networking", core.InstanceStatusUpdated)
	updatedParent := instance("parent-1", "orders", core.InstanceStatusUpdated)
	updatedParent.ChildBlueprints = map[string]*state.InstanceState{"networking": &updatedChild}
	s.Require().NoError(s.importInstances(false, updatedParent))

	storedChild, err := s.storedInstance("child-1")
	s.Require().NoError(err)
	s.Equal(core.InstanceStatusUpdated, storedChild.Status)
	storedParent, err := s.storedInstance("parent-1")
	s.Require().NoError(err)
	s.Require().Contains(storedParent.ChildBlueprints, "networking")
	s.Equal(core.InstanceStatusUpdated, storedParent.ChildBlueprints["networking"].Status)
}

func (s *ImportChecksTestSuite) Test_checks_nested_children_too() {
	s.Require().NoError(s.importInstances(false, instance("child-1", "orders-networking", core.InstanceStatusUpdating)))

	child := instance("child-1", "orders-networking", core.InstanceStatusDeployed)
	parent := instance("parent-1", "orders", core.InstanceStatusDeployed)
	parent.ChildBlueprints = map[string]*state.InstanceState{"networking": &child}
	err := s.importInstances(false, parent)

	importErr := s.assertImportError(err, ErrCodeInstanceOperationInProgress)
	s.Equal(
		"nothing was imported\ninstances with an operation in progress:\n"+
			`  - child-1 ("orders-networking") is UPDATING`,
		importErr.Message,
	)
	_, err = s.storedInstance("parent-1")
	s.True(state.IsInstanceNotFound(err), "expected the parent not to be written")
}

func (s *ImportChecksTestSuite) Test_reports_every_problem_at_once() {
	s.Require().NoError(s.importInstances(
		false,
		instance("inst-1", "orders", core.InstanceStatusDeployed),
		instance("inst-2", "billing", core.InstanceStatusDeploying),
	))

	err := s.importInstances(
		false,
		instance("inst-1", "orders-v2", core.InstanceStatusDeployed),
		instance("inst-2", "billing", core.InstanceStatusDeployed),
	)

	importErr := s.assertImportError(err, ErrCodeInstanceNameImmutable)
	s.Equal(
		"nothing was imported\ninstances cannot be renamed:\n"+
			`  - inst-1 is named "orders", not "orders-v2"`+"\n"+
			"instances with an operation in progress:\n"+
			`  - inst-2 ("billing") is DEPLOYING`,
		importErr.Message,
	)
}

func TestImportChecksTestSuite(t *testing.T) {
	suite.Run(t, new(ImportChecksTestSuite))
}
