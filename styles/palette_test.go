package styles

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/suite"
)

type PaletteSuite struct {
	suite.Suite
}

// Palette name tests

func (s *PaletteSuite) Test_bluelink_palette_name() {
	p := NewBluelinkPalette()
	s.Equal("bluelink", p.Name())
}

func (s *PaletteSuite) Test_celerity_palette_name() {
	p := NewCelerityPalette()
	s.Equal("celerity", p.Name())
}

// Color differentiation tests

func (s *PaletteSuite) Test_palettes_have_different_primary_colors() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.NotEqual(bluelink.Primary(), celerity.Primary())
}

func (s *PaletteSuite) Test_palettes_have_different_secondary_colors() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.NotEqual(bluelink.Secondary(), celerity.Secondary())
}

// Shared semantic colors tests

func (s *PaletteSuite) Test_palettes_share_error_color() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.Equal(bluelink.Error(), celerity.Error())
}

func (s *PaletteSuite) Test_palettes_share_warning_color() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.Equal(bluelink.Warning(), celerity.Warning())
}

func (s *PaletteSuite) Test_palettes_share_info_color() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.Equal(bluelink.Info(), celerity.Info())
}

func (s *PaletteSuite) Test_palettes_share_success_color() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.Equal(bluelink.Success(), celerity.Success())
}

func (s *PaletteSuite) Test_palettes_share_muted_color() {
	bluelink := NewBluelinkPalette()
	celerity := NewCelerityPalette()

	s.Equal(bluelink.Muted(), celerity.Muted())
}

// NewStyles tests

func (s *PaletteSuite) Test_new_styles_with_bluelink_palette() {
	r := lipgloss.NewRenderer(os.Stdout)
	palette := NewBluelinkPalette()
	styles := NewStyles(r, palette)

	s.NotNil(styles)
	s.Equal(palette, styles.Palette)
	s.Equal("bluelink", styles.Palette.Name())
}

func (s *PaletteSuite) Test_new_styles_with_celerity_palette() {
	r := lipgloss.NewRenderer(os.Stdout)
	palette := NewCelerityPalette()
	styles := NewStyles(r, palette)

	s.NotNil(styles)
	s.Equal(palette, styles.Palette)
	s.Equal("celerity", styles.Palette.Name())
}

func (s *PaletteSuite) Test_new_default_styles_with_palette() {
	palette := NewBluelinkPalette()
	styles := NewDefaultStyles(palette)

	s.NotNil(styles)
	s.Equal(palette, styles.Palette)
}

// Huh theme tests

func (s *PaletteSuite) Test_new_huh_theme_with_palette() {
	palette := NewBluelinkPalette()
	theme := NewHuhTheme(palette)

	s.NotNil(theme)
	s.NotNil(theme.Focused)
	s.NotNil(theme.Blurred)
}

func (s *PaletteSuite) Test_new_bluelink_huh_theme_backwards_compat() {
	theme := NewBluelinkHuhTheme()

	s.NotNil(theme)
}

// Style rendering tests

func (s *PaletteSuite) Test_styles_render_text() {
	styles := NewDefaultStyles(NewBluelinkPalette())

	s.NotEmpty(styles.Selected.Render("test"))
	s.NotEmpty(styles.Title.Render("test"))
	s.NotEmpty(styles.Error.Render("test"))
	s.NotEmpty(styles.Warning.Render("test"))
	s.NotEmpty(styles.Info.Render("test"))
	s.NotEmpty(styles.Muted.Render("test"))
	s.NotEmpty(styles.Spinner.Render("test"))
	s.NotEmpty(styles.Category.Render("test"))
}

func TestPaletteSuite(t *testing.T) {
	suite.Run(t, new(PaletteSuite))
}
