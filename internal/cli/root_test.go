package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.NotEmpty(t, buf.String())
}

func TestDoctorWithExampleConfig(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	configPath, err := filepath.Abs(filepath.Join("..", "..", "configs", "config.example.yaml"))
	require.NoError(t, err)
	require.FileExists(t, configPath)

	cmd.SetArgs([]string{"doctor", "--config", configPath})

	err = cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, buf.String(), "Config OK")
}
