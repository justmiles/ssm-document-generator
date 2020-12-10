# SSM Document Generator

Build and maintain SSM documents.

## Usage

```bash
curl https://gobinaries.com/justmiles/ssm-document-generator | sh
ssm-document-generator <path to template>
```

## Templates

The templates are a yaml version of an [SSM document syntax](https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-doc-syntax.html). The key difference is the added `runCommandScript` which is used to generate the `runCommand` array.

```yaml
schemaVersion: "2.2"
description: Runs a custom script or workload.
parameters: {}
mainSteps:
  - precondition:
      StringEquals:
        - platformType
        - Windows
    action: aws:runPowerShellScript
    name: ScanWindows
    inputs:
      timeoutSeconds: 7200
      runCommandScript: ../scripts/my-custom-script.ps1

  - precondition:
      StringEquals:
        - platformType
        - Linux
    action: aws:runShellScript
    name: ScanLinux
    inputs:
      timeoutSeconds: 7200
      runCommandScript: ../scripts/my-custom-script.sh

  - precondition:
      StringEquals:
        - platformType
        - MacOS
    action: aws:runShellScript
    name: ScanMacOS
    inputs:
      timeoutSeconds: 7200


```