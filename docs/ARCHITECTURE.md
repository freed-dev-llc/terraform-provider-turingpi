# Architecture Documentation

This document provides visual architecture documentation for the Turing Pi Terraform Provider.

## Table of Contents

1. [System Overview](#system-overview)
2. [Network Topology](#network-topology)
3. [Component Interaction](#component-interaction)
4. [Resource Lifecycle](#resource-lifecycle)
5. [Data Flow](#data-flow)
6. [Authentication Flow](#authentication-flow)

---

## System Overview

```mermaid
graph TB
    subgraph "Terraform Workflow"
        TF[Terraform CLI]
        TFState[(terraform.tfstate)]
        TFConfig[main.tf Configuration]
    end

    subgraph "Provider Layer"
        Provider[terraform-provider-turingpi]
        Auth[auth.go<br/>Token Management]
        Helpers[helpers.go<br/>API Helpers]
    end

    subgraph "Resources"
        RPower[turingpi_power<br/>Node Power Control]
        RFlash[turingpi_flash<br/>Firmware Flashing]
        RNode[turingpi_node<br/>Full Provisioning]
    end

    subgraph "Turing Pi BMC"
        BMC[BMC API<br/>https://turingpi.local]
        UART[UART Interface<br/>Boot Monitoring]
        Nodes[Compute Nodes 1-4]
    end

    TFConfig --> TF
    TF <--> TFState
    TF --> Provider
    Provider --> Auth
    Provider --> Helpers
    Provider --> RPower
    Provider --> RFlash
    Provider --> RNode

    Auth --> BMC
    Helpers --> BMC
    RPower --> BMC
    RFlash --> BMC
    RNode --> BMC
    RNode --> UART
    BMC --> Nodes
```

---

## Network Topology

```mermaid
graph TB
    subgraph "Operator Workstation"
        TF[Terraform CLI]
    end

    subgraph "Network Layer"
        HTTPS[HTTPS/TLS<br/>Port 443]
    end

    subgraph "Turing Pi 2.5 Board"
        subgraph "BMC"
            API[REST API<br/>/api/bmc/*]
            AUTH[Authentication<br/>/api/bmc/authenticate]
            UARTAPI[UART Endpoint<br/>/api/bmc?opt=get&type=uart]
        end

        subgraph "Compute Modules"
            N1[Node 1<br/>Slot 1]
            N2[Node 2<br/>Slot 2]
            N3[Node 3<br/>Slot 3]
            N4[Node 4<br/>Slot 4]
        end

        subgraph "BMC Hardware"
            PWR[Power Control]
            FLASH[Flash Controller]
            UART1[UART 1]
            UART2[UART 2]
            UART3[UART 3]
            UART4[UART 4]
        end
    end

    TF --> HTTPS
    HTTPS --> AUTH
    HTTPS --> API
    HTTPS --> UARTAPI

    API --> PWR
    API --> FLASH
    UARTAPI --> UART1
    UARTAPI --> UART2
    UARTAPI --> UART3
    UARTAPI --> UART4

    PWR --> N1
    PWR --> N2
    PWR --> N3
    PWR --> N4

    FLASH --> N1
    FLASH --> N2
    FLASH --> N3
    FLASH --> N4

    UART1 --> N1
    UART2 --> N2
    UART3 --> N3
    UART4 --> N4
```

---

## Component Interaction

```mermaid
graph LR
    subgraph "Terraform Plugin SDK v2"
        Plugin[plugin.Serve]
        Schema[Provider Schema]
    end

    subgraph "provider/provider.go"
        ProviderFunc[Provider Function]
        ConfigFunc[Configure Function]
        ProviderConfig[ProviderConfig Struct]
    end

    subgraph "provider/auth.go"
        Authenticate[Authenticate Function]
        Token[(Bearer Token)]
    end

    subgraph "provider/helpers.go"
        PowerOn[turnOnNode]
        PowerOff[turnOffNode]
        FlashFW[initFlash]
        CheckBoot[waitForBootCompletion]
        CheckPower[checkPowerStatus]
    end

    subgraph "Resource Files"
        ResPower[resource_power.go]
        ResFlash[resource_flash.go]
        ResNode[resource_node.go]
    end

    Plugin --> ProviderFunc
    ProviderFunc --> Schema
    Schema --> ConfigFunc
    ConfigFunc --> Authenticate
    Authenticate --> Token
    Token --> ProviderConfig

    ProviderConfig --> ResPower
    ProviderConfig --> ResFlash
    ProviderConfig --> ResNode

    ResPower --> PowerOn
    ResPower --> PowerOff
    ResPower --> CheckPower

    ResFlash --> FlashFW

    ResNode --> PowerOn
    ResNode --> PowerOff
    ResNode --> FlashFW
    ResNode --> CheckBoot
```

---

## Resource Lifecycle

### turingpi_power Resource

```mermaid
stateDiagram-v2
    [*] --> Create: terraform apply
    Create --> Read: Success
    Create --> [*]: Error

    Read --> Update: State Change Detected
    Read --> Delete: terraform destroy
    Read --> Read: No Changes

    Update --> Read: Success
    Update --> [*]: Error

    Delete --> [*]: Success/Error

    state Create {
        [*] --> SetPowerState
        SetPowerState --> SetResourceID
        SetResourceID --> [*]
    }

    state Update {
        [*] --> CheckCurrentState
        CheckCurrentState --> TogglePower
        TogglePower --> [*]
    }
```

### turingpi_flash Resource

```mermaid
stateDiagram-v2
    [*] --> Create: terraform apply
    Create --> Read: Success
    Create --> [*]: Error

    Read --> ForceNew: firmware_file changed
    Read --> ForceNew: node changed
    Read --> Delete: terraform destroy

    ForceNew --> Delete: Destroy Old
    Delete --> Create: Recreate

    Delete --> [*]: Success

    state Create {
        [*] --> ValidateFirmware
        ValidateFirmware --> InitFlash
        InitFlash --> WaitForCompletion
        WaitForCompletion --> SetResourceID
        SetResourceID --> [*]
    }
```

### turingpi_node Resource (Composite)

```mermaid
stateDiagram-v2
    [*] --> Create: terraform apply

    state Create {
        [*] --> PowerControl
        PowerControl --> FlashFirmware: firmware_file set
        PowerControl --> BootCheck: No firmware
        FlashFirmware --> BootCheck: boot_check enabled
        FlashFirmware --> Complete: No boot_check
        BootCheck --> Complete: Pattern found
        BootCheck --> Timeout: Timeout exceeded
        Complete --> [*]
        Timeout --> [*]: Error
    }

    Create --> Read: Success
    Read --> Delete: terraform destroy

    state Delete {
        [*] --> PowerOff
        PowerOff --> [*]
    }

    Delete --> [*]
```

---

## Data Flow

### Provisioning Workflow

```mermaid
sequenceDiagram
    participant User
    participant Terraform
    participant Provider
    participant BMC
    participant Node

    User->>Terraform: terraform apply
    Terraform->>Provider: Configure Provider
    Provider->>BMC: POST /api/bmc/authenticate
    BMC-->>Provider: Bearer Token

    alt turingpi_node resource
        Provider->>BMC: Power On Node
        BMC->>Node: Enable Power
        Node-->>BMC: Power OK

        opt firmware_file specified
            Provider->>BMC: Flash Firmware
            BMC->>Node: Write Firmware
            Node-->>BMC: Flash Complete
        end

        opt boot_check enabled
            loop Until Pattern Found or Timeout
                Provider->>BMC: GET /api/bmc?type=uart&node=N
                BMC->>Node: Read UART
                Node-->>BMC: UART Output
                BMC-->>Provider: UART Response
                Provider->>Provider: Check for login prompt
            end
        end
    end

    Provider-->>Terraform: Resource Created
    Terraform-->>User: Apply Complete
```

### Boot Verification Detail

```mermaid
sequenceDiagram
    participant Provider
    participant BMC
    participant UART
    participant Node

    Provider->>Provider: Start boot_check timer

    loop Poll UART (default: login_prompt_timeout)
        Provider->>BMC: GET /api/bmc?opt=get&type=uart&node={id}
        BMC->>UART: Read Buffer
        UART->>Node: Serial RX
        Node-->>UART: Boot Messages
        UART-->>BMC: UART Content
        BMC-->>Provider: JSON Response

        alt Pattern Found
            Provider->>Provider: Match "login:" or custom pattern
            Provider-->>Provider: Return Success
        else Timeout
            Provider-->>Provider: Return Error with last output
        end
    end
```

---

## Authentication Flow

```mermaid
sequenceDiagram
    participant TF as Terraform
    participant Provider
    participant Auth as auth.go
    participant BMC as BMC API

    TF->>Provider: Initialize Provider
    Provider->>Auth: Authenticate(endpoint, username, password)

    Auth->>BMC: POST /api/bmc/authenticate
    Note over Auth,BMC: Body: {"username": "...", "password": "..."}

    alt Success
        BMC-->>Auth: 200 OK + Token
        Auth-->>Provider: token string
        Provider->>Provider: Store in ProviderConfig.Token
    else Failure
        BMC-->>Auth: 401 Unauthorized
        Auth-->>Provider: Error
        Provider-->>TF: Configuration Error
    end

    Note over Provider: Token used in all subsequent requests
    Provider->>BMC: GET /api/bmc/...
    Note over Provider,BMC: Header: Authorization: Bearer {token}
```

---

## API Endpoints Reference

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/api/bmc/authenticate` | POST | Get bearer token | None |
| `/api/bmc?opt=get&type=uart&node={id}` | GET | Read UART output | Bearer |
| `/api/bmc?opt=set&type=power&node={id}&mode={0\|1}` | GET | Power control | Bearer |
| `/api/bmc?opt=set&type=flash&node={id}` | POST | Flash firmware | Bearer |

---

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `TURINGPI_USERNAME` | BMC username | - |
| `TURINGPI_PASSWORD` | BMC password | - |
| `TURINGPI_ENDPOINT` | BMC URL | `https://turingpi.local` |
| `TURINGPI_INSECURE` | Skip TLS verification | `false` |

---

## File Structure

```
terraform-provider-turingpi/
├── main.go                              # Plugin entry point
├── provider/
│   ├── provider.go                      # Provider schema and config
│   ├── auth.go                          # Authentication logic
│   ├── helpers.go                       # Shared API helpers
│   ├── cluster_helpers.go               # Cluster provisioning utilities
│   ├── ssh_client.go                    # SSH connection management
│   ├── helm_client.go                   # Helm chart deployment
│   ├── k8s_client.go                    # Kubernetes manifest client
│   ├── kubeconfig.go                    # Kubeconfig utilities
│   ├── k3s_provisioner.go              # K3s installation logic
│   ├── talos_provisioner.go            # Talos provisioning via talosctl
│   ├── data_source_about.go            # BMC version info
│   ├── data_source_info.go             # BMC info (version, network, storage)
│   ├── data_source_power.go            # Node power status
│   ├── data_source_sdcard.go           # SD card storage info
│   ├── data_source_uart.go             # UART serial output
│   ├── data_source_usb.go             # USB routing config
│   ├── resource_bmc_firmware.go        # BMC firmware upgrade
│   ├── resource_bmc_reboot.go          # BMC reboot
│   ├── resource_bmc_reload.go          # BMC daemon reload
│   ├── resource_clear_usb_boot.go      # Clear USB boot status
│   ├── resource_flash.go               # Firmware flash
│   ├── resource_k3s_cluster.go         # K3s cluster (deprecated)
│   ├── resource_network_reset.go       # Network switch reset
│   ├── resource_node.go                # Combined node management
│   ├── resource_node_to_msd.go         # USB Mass Storage mode
│   ├── resource_power.go               # Node power control
│   ├── resource_talos_cluster.go       # Talos cluster (deprecated)
│   ├── resource_uart.go                # UART write
│   ├── resource_usb.go                 # USB routing
│   └── resource_usb_boot.go            # USB boot mode
├── docs/
│   ├── index.md                         # Registry documentation
│   ├── data-sources/                    # Data source documentation
│   └── resources/                       # Resource documentation
├── examples/                            # Usage examples
└── .goreleaser.yml                      # Release configuration
```
