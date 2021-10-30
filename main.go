package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// Document ..
type Document struct {
	SchemaVersion string `yaml:"schemaVersion" json:"schemaVersion"`
	Description   string `yaml:"description" json:"description"`
	Parameters    map[string]struct {
		Type        string `yaml:"type,omitempty" json:"type"`
		Description string `yaml:"description,omitempty" json:"description"`
		Default     string `yaml:"default,omitempty" json:"default"`
	} `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	MainSteps []struct {
		Precondition struct {
			StringEquals []string `yaml:"StringEquals,omitempty" json:"StringEquals,omitempty"`
		} `yaml:"precondition,omitempty" json:"precondition,omitempty"`
		Action string `yaml:"action" json:"action"`
		Name   string `yaml:"name" json:"name"`
		Inputs struct {
			TimeoutSeconds     int      `yaml:"timeoutSeconds" json:"timeoutSeconds"`
			RunCommand         []string `yaml:"runCommand,omitempty" json:"runCommand,omitempty"`
			RunCommandScript   string   `yaml:"runCommandScript,omitempty" json:"runCommandScript,omitempty"`
			DocumentType       string   `yaml:"documentType,omitempty" json:"documentType,omitempty"`
			DocumentPath       string   `yaml:"documentPath,omitempty" json:"documentPath,omitempty"`
			DocumentParameters string   `yaml:"documentParameters,omitempty" json:"documentParameters,omitempty"`
		} `yaml:"inputs" json:"inputs"`
	} `yaml:"mainSteps" json:"mainSteps"`
}

func main() {

	var (
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		ssmSvc     = ssm.New(sess)
		file, name string
		document   Document
	)

	if len(os.Args) > 1 {
		file = os.Args[1]
		name = strings.ReplaceAll(path.Base(file), ".yaml", "")
	}

	dat, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("Error reading '%s': %s\n", file, err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(dat, &document)
	if err != nil {
		fmt.Printf("Error parsing YAML document '%s': %s\n", file, err)
		os.Exit(1)
	}

	for i, step := range document.MainSteps {
		if step.Inputs.RunCommandScript != "" {
			dat, err := ioutil.ReadFile(path.Join(path.Dir(file), step.Inputs.RunCommandScript))
			if err != nil {
				fmt.Printf("Error reading script '%s': %s\n", step.Inputs.RunCommandScript, err)
				os.Exit(1)
			}
			document.MainSteps[i].Inputs.RunCommand = strings.Split(string(dat), "\n")
			document.MainSteps[i].Inputs.RunCommandScript = ""
		}
	}

	err = document.create(ssmSvc, name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (d *Document) create(s *ssm.SSM, name string) error {

	for i, step := range d.MainSteps {
		if len(step.Precondition.StringEquals) == 0 {
			fmt.Println(d.MainSteps[i].Precondition)
		}
	}

	dat, err := json.Marshal(d)
	if err != nil {
		return err
	}

	_, err = s.CreateDocument(&ssm.CreateDocumentInput{
		Name:           &name,
		Content:        aws.String(string(dat)),
		DocumentFormat: aws.String("JSON"),
		DocumentType:   aws.String("Command"),
		TargetType:     aws.String("/"),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			if awsErr.Code() == ssm.ErrCodeDocumentAlreadyExists {
				return d.update(s, name)
			}
			return err
		}
		return err
	}

	fmt.Printf("created %s\n", name)
	return err
}

func (d *Document) update(s *ssm.SSM, name string) error {
	dat, err := json.Marshal(d)
	if err != nil {
		return err
	}

	o, err := s.UpdateDocument(&ssm.UpdateDocumentInput{
		Name:            &name,
		Content:         aws.String(string(dat)),
		DocumentFormat:  aws.String("JSON"),
		TargetType:      aws.String("/"),
		DocumentVersion: aws.String("$LATEST"),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			if awsErr.Code() == ssm.ErrCodeDuplicateDocumentContent {
				fmt.Println("No changes to document.")
				return nil
			}
			return err
		}
		return err
	}

	fmt.Printf("updated %s\n", name)
	updateDocumentDefaultVersionOutput, err := s.UpdateDocumentDefaultVersion(&ssm.UpdateDocumentDefaultVersionInput{
		Name:            &name,
		DocumentVersion: o.DocumentDescription.DocumentVersion,
	})

	fmt.Println(updateDocumentDefaultVersionOutput.Description)
	return err
}
