package kingpin

import (
	"io/ioutil"
	"os"

	"github.com/alecthomas/assert"

	"testing"
)

func TestBool(t *testing.T) {
	app := newTestApp()
	b := app.Flag("b", "").Bool()
	_, err := app.Parse([]string{"--b"})
	assert.NoError(t, err)
	assert.True(t, *b)
}

func TestNoBool(t *testing.T) {
	fg := newFlagGroup()
	f := fg.Flag("b", "").Default("true")
	b := f.Bool()
	fg.init("")
	tokens := tokenize([]string{"--no-b"}, false)
	_, err := fg.parse(tokens)
	assert.NoError(t, err)
	assert.False(t, *b)
}

func TestNegateNonBool(t *testing.T) {
	fg := newFlagGroup()
	f := fg.Flag("b", "")
	f.Int()
	fg.init("")
	tokens := tokenize([]string{"--no-b"}, false)
	_, err := fg.parse(tokens)
	assert.Error(t, err)
}

func TestInvalidFlagDefaultCanBeOverridden(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").Default("invalid").Bool()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestRequiredFlag(t *testing.T) {
	app := newTestApp()
	app.Version("0.0.0").Writer(ioutil.Discard)
	exits := 0
	app.Terminate(func(int) { exits++ })
	app.Flag("a", "").Required().Bool()
	_, err := app.Parse([]string{"--a"})
	assert.NoError(t, err)
	_, err = app.Parse([]string{})
	assert.Error(t, err)
	_, err = app.Parse([]string{"--version"})
	assert.Equal(t, 1, exits)
}

func TestShortFlag(t *testing.T) {
	app := newTestApp()
	f := app.Flag("long", "").Short('s').Bool()
	_, err := app.Parse([]string{"-s"})
	assert.NoError(t, err)
	assert.True(t, *f)
}

func TestCombinedShortFlags(t *testing.T) {
	app := newTestApp()
	a := app.Flag("short0", "").Short('0').Bool()
	b := app.Flag("short1", "").Short('1').Bool()
	c := app.Flag("short2", "").Short('2').Bool()
	_, err := app.Parse([]string{"-01"})
	assert.NoError(t, err)
	assert.True(t, *a)
	assert.True(t, *b)
	assert.False(t, *c)
}

func TestCombinedShortFlagArg(t *testing.T) {
	a := newTestApp()
	n := a.Flag("short", "").Short('s').Int()
	_, err := a.Parse([]string{"-s10"})
	assert.NoError(t, err)
	assert.Equal(t, 10, *n)
}

func TestEmptyShortFlagIsAnError(t *testing.T) {
	_, err := newTestApp().Parse([]string{"-"})
	assert.Error(t, err)
}

func TestRequiredWithEnvarMissingErrors(t *testing.T) {
	app := newTestApp()
	app.Flag("t", "").OverrideDefaultFromEnvar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	flag := app.Flag("t", "").Envar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestSubcommandFlagRequiredWithEnvar(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	cmd := app.Command("command", "")
	flag := cmd.Flag("t", "").Envar("TEST_ENVAR").Required().Int()
	_, err := app.Parse([]string{"command"})
	assert.NoError(t, err)
	assert.Equal(t, 123, *flag)
}

func TestRegexp(t *testing.T) {
	app := newTestApp()
	flag := app.Flag("reg", "").Regexp()
	_, err := app.Parse([]string{"--reg", "^abc$"})
	assert.NoError(t, err)
	assert.NotNil(t, *flag)
	assert.Equal(t, "^abc$", (*flag).String())
	assert.Regexp(t, *flag, "abc")
	assert.NotRegexp(t, *flag, "abcd")
}

func TestDuplicateShortFlag(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").Short('a').String()
	app.Flag("b", "").Short('a').String()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestDuplicateLongFlag(t *testing.T) {
	app := newTestApp()
	app.Flag("a", "").String()
	app.Flag("a", "").String()
	_, err := app.Parse([]string{})
	assert.Error(t, err)
}

func TestGetFlagAndOverrideDefault(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Default("default").String()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "default", *a)
	app.GetFlag("a").Default("new")
	_, err = app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "new", *a)
}

func TestEnvarOverrideDefault(t *testing.T) {
	os.Setenv("TEST_ENVAR", "123")
	app := newTestApp()
	flag := app.Flag("t", "").Default("default").Envar("TEST_ENVAR").String()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "123", *flag)
}

func TestFlagMultipleValuesDefault(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Default("default1", "default2").Strings()
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"default1", "default2"}, *a)
}

func TestFlagMultipleValuesDefaultNonRepeatable(t *testing.T) {
	c := newTestApp()
	c.Flag("foo", "foo").Default("a", "b").String()
	_, err := c.Parse([]string{})
	assert.Error(t, err)
}

func TestFlagMultipleValuesDefaultEnvarUnix(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Envar("TEST_MULTIPLE_VALUES").Strings()
	os.Setenv("TEST_MULTIPLE_VALUES", "123\n456\n")
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "456"}, *a)
}

func TestFlagMultipleValuesDefaultEnvarWindows(t *testing.T) {
	app := newTestApp()
	a := app.Flag("a", "").Envar("TEST_MULTIPLE_VALUES").Strings()
	os.Setenv("TEST_MULTIPLE_VALUES", "123\r\n456\r\n")
	_, err := app.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "456"}, *a)
}

func TestFlagMultipleValuesDefaultEnvarNonRepeatable(t *testing.T) {
	c := newTestApp()
	a := c.Flag("foo", "foo").Envar("TEST_MULTIPLE_VALUES_NON_REPEATABLE").String()
	os.Setenv("TEST_MULTIPLE_VALUES_NON_REPEATABLE", "123\n456")
	_, err := c.Parse([]string{})
	assert.NoError(t, err)
	assert.Equal(t, "123\n456", *a)
}
