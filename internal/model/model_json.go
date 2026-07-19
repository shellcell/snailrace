package model

import (
	"encoding/json"
	"path/filepath"
)

func (tool *ToolInfo) UnmarshalJSON(data []byte) error {
	type plain ToolInfo
	decoded := struct {
		plain
		ShellCommand       *bool `json:"shell_command"`
		ProvenanceVerified *bool `json:"provenance_verified"`
	}{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*tool = ToolInfo(decoded.plain)
	if decoded.ShellCommand != nil {
		tool.ShellCommand = *decoded.ShellCommand
	} else {
		tool.ShellCommand = len(tool.Command) == 1 &&
			tool.Command[0] != tool.Executable &&
			tool.Command[0] != filepath.Base(tool.Executable)
	}
	if decoded.ProvenanceVerified != nil {
		tool.ProvenanceVerified = *decoded.ProvenanceVerified
	} else {
		tool.ProvenanceVerified = tool.SHA256 != ""
	}
	return nil
}

func (run *Run) UnmarshalJSON(data []byte) error {
	type plain Run
	decoded := struct {
		plain
		PhysicalFootprintValid *bool `json:"physical_footprint_valid"`
	}{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*run = Run(decoded.plain)
	if decoded.PhysicalFootprintValid != nil {
		run.PhysicalFootprintValid = *decoded.PhysicalFootprintValid
	} else {
		run.PhysicalFootprintValid = run.PeakPhysicalFootprintBytes > 0
	}
	return nil
}
