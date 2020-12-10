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

func main() {

	var (
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		ssmSvc   = ssm.New(sess)
		file     = os.Args[1]
		name     = strings.ReplaceAll(path.Base(file), ".yaml", "")
		document Document
	)

	dat, err := ioutil.ReadFile(file)
	check(err)

	err = yaml.Unmarshal(dat, &document)
	check(err)

	for i, step := range document.MainSteps {
		if step.Inputs.RunCommandScript != "" {
			dat, err := ioutil.ReadFile(path.Join(path.Dir(file), step.Inputs.RunCommandScript))
			check(err)
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

// Document ..
type Document struct {
	SchemaVersion string `yaml:"schemaVersion" json:"schemaVersion"`
	Description   string `yaml:"description" json:"description"`
	Parameters    struct {
	} `yaml:"parameters" json:"parameters"`
	MainSteps []struct {
		Precondition struct {
			StringEquals []string `yaml:"StringEquals" json:"StringEquals"`
		} `yaml:"precondition" json:"precondition"`
		Action string `yaml:"action" json:"action"`
		Name   string `yaml:"name" json:"name"`
		Inputs struct {
			TimeoutSeconds   int      `yaml:"timeoutSeconds" json:"timeoutSeconds"`
			RunCommand       []string `yaml:"runCommand" json:"runCommand"`
			RunCommandScript string   `yaml:"runCommandScript,omitempty" json:"runCommandScript,omitempty"`
		} `yaml:"inputs" json:"inputs"`
	} `yaml:"mainSteps" json:"mainSteps"`
}

func (d *Document) create(s *ssm.SSM, name string) error {
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
		}
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
				return nil
			}
		}
	}

	fmt.Printf("updated %s\n", name)
	_, err = s.UpdateDocumentDefaultVersion(&ssm.UpdateDocumentDefaultVersionInput{
		Name:            &name,
		DocumentVersion: o.DocumentDescription.DocumentVersion,
	})
	return err

}

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
