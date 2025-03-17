// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"
	"text/template"
)

type TemplateData struct {
	ProcessName     string
	ProcessAlias    string
	ThreadCount     int
	HalfThreadCount int
	TPS             int
	FileCount       int
	Duration        string
}

func processTemplate(templatePath string, outputPath string, templateData TemplateData) error {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err)
	}
	output, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening output file: %v", err)
	}
	defer output.Close()
	if err = tmpl.Execute(output, templateData); err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}
	return nil
}
