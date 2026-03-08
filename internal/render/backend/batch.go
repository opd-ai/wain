package backend

import (
	"github.com/opd-ai/wain/internal/raster/displaylist"
)

// PipelineType represents a GPU pipeline state.
type PipelineType uint8

const (
	// PipelineSolidFill is the solid color fill pipeline.
	PipelineSolidFill PipelineType = iota

	// PipelineTextured is the textured quad pipeline.
	PipelineTextured

	// PipelineText is the SDF text rendering pipeline.
	PipelineText

	// PipelineLinearGradient is the linear gradient pipeline.
	PipelineLinearGradient

	// PipelineRadialGradient is the radial gradient pipeline.
	PipelineRadialGradient

	// PipelineBoxShadow is the box shadow pipeline.
	PipelineBoxShadow
)

// Batch represents a group of draw commands that share the same pipeline state.
type Batch struct {
	Pipeline PipelineType
	Commands []displaylist.DrawCommand
}

// batchCommands sorts and groups commands by pipeline state to minimize GPU state changes.
func batchCommands(commands []displaylist.DrawCommand) []Batch {
	if len(commands) == 0 {
		return nil
	}

	// Group commands by pipeline type
	batches := make([]Batch, 0, 16)
	var currentBatch *Batch

	for _, cmd := range commands {
		pipeline := commandToPipeline(cmd.Type)

		// Start a new batch if pipeline changed or this is the first command
		if currentBatch == nil || currentBatch.Pipeline != pipeline {
			batches = append(batches, Batch{
				Pipeline: pipeline,
				Commands: make([]displaylist.DrawCommand, 0, 32),
			})
			currentBatch = &batches[len(batches)-1]
		}

		currentBatch.Commands = append(currentBatch.Commands, cmd)
	}

	return batches
}

// commandToPipeline maps a command type to its corresponding pipeline.
func commandToPipeline(cmdType displaylist.CommandType) PipelineType {
	switch cmdType {
	case displaylist.CmdFillRect, displaylist.CmdFillRoundedRect, displaylist.CmdDrawLine:
		return PipelineSolidFill
	case displaylist.CmdDrawImage:
		return PipelineTextured
	case displaylist.CmdDrawText:
		return PipelineText
	case displaylist.CmdLinearGradient:
		return PipelineLinearGradient
	case displaylist.CmdRadialGradient:
		return PipelineRadialGradient
	case displaylist.CmdBoxShadow:
		return PipelineBoxShadow
	default:
		return PipelineSolidFill
	}
}
