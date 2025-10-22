package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alecthomas/jsonschema"
	"github.com/concourse/concourse/atc"
)

func stepSchema(schema *jsonschema.Schema) {
	stepDef := schema.Definitions["Step"]

	// thx: https://gist.github.com/4662752c4af75b3220d94b657683d090
	// https://concourse-ci.org/steps.html#schema.step

	// Core step types that can appear in the anyOf
	coreStepConfigs := []atc.StepConfig{
		&atc.GetStep{},
		&atc.PutStep{},
		&atc.TaskStep{},
		&atc.SetPipelineStep{},
		&atc.LoadVarStep{},
		&atc.InParallelStep{},
		&atc.DoStep{},
		&atc.TryStep{},
	}

	// Hook and modifier properties that can be added to any step
	hookAndModifierConfigs := []atc.StepConfig{
		&atc.OnSuccessStep{},
		&atc.OnFailureStep{},
		&atc.OnAbortStep{},
		&atc.OnErrorStep{},
		&atc.EnsureStep{},
		&atc.TimeoutStep{},
		&atc.RetryStep{},
		&atc.AcrossStep{},
	}

	// Add core step types to anyOf
	for _, s := range coreStepConfigs {
		stepSchema := jsonschema.Reflect(s)
		stepDef.AnyOf = append(stepDef.AnyOf, stepSchema.Type)
		for subName, subDef := range stepSchema.Definitions {
			if _, present := schema.Definitions[subName]; !present {
				schema.Definitions[subName] = subDef
			}
		}
	}

	// Reflect hook and modifier types to get their definitions
	for _, s := range hookAndModifierConfigs {
		stepSchema := jsonschema.Reflect(s)
		for subName, subDef := range stepSchema.Definitions {
			if _, present := schema.Definitions[subName]; !present {
				schema.Definitions[subName] = subDef
			}
		}
	}

	// Add hook and modifier properties to each core step type
	// These hooks can be applied to any step type
	hookProperties := map[string]*jsonschema.Type{
		"on_success": {Ref: "#/definitions/Step"},
		"on_failure": {Ref: "#/definitions/Step"},
		"on_abort":   {Ref: "#/definitions/Step"},
		"on_error":   {Ref: "#/definitions/Step"},
		"ensure":     {Ref: "#/definitions/Step"},
		"timeout":    {Type: "string"},
		"attempts":   {Type: "integer"},
		"across":     {Type: "array", Items: &jsonschema.Type{Ref: "#/definitions/AcrossVarConfig"}},
		"fail_fast":  {Type: "boolean"},
	}

	// Add hook properties to each core step definition
	for _, stepName := range []string{"GetStep", "PutStep", "TaskStep", "SetPipelineStep", "LoadVarStep", "InParallelStep", "DoStep", "TryStep"} {
		if stepTypeDef, exists := schema.Definitions[stepName]; exists {
			for hookName, hookType := range hookProperties {
				stepTypeDef.Properties.Set(hookName, hookType)
			}
		}
	}

	stepDef.Required = nil
	schema.Definitions["Step"].Properties = nil
	schema.Definitions["Step"].Type = ""
	schema.Definitions["Step"].AdditionalProperties = nil
}

func main() {
	var pipelineConfig atc.Config
	schema := jsonschema.Reflect(&pipelineConfig)
	stepSchema(schema)
	schema.AdditionalProperties = json.RawMessage("true")
	schema.Definitions["CheckEvery"].Type = "string"
	reflected, err := schema.MarshalJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(reflected))
}
