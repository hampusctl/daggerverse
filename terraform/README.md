# Terraform Dagger Module

En Dagger-modul för att köra Terraform-kommandon i en containeriserad miljö.

## Användning

### Skapa en plan och spara som fil

```bash
# Skapar output.tfplan fil
dagger call new --source . plan export --path ./output.tfplan

# Med extra argument (t.ex. target specifik resurs)
dagger call new --source . plan --extra-args="-target=aws_instance.example" export --path ./output.tfplan
```

### Applicera en specifik plan-fil

```bash
# Applicerar en sparad plan-fil med auto-approve
dagger call new --source . apply --plan-file ./output.tfplan

# Med extra argument
dagger call new --source . apply --plan-file ./output.tfplan --extra-args="-parallelism=5"
```

### Initialisera med extra argument

```bash
# Init med backend-config
dagger call new --source . init --extra-args="-backend-config=bucket=my-terraform-state"
```

### Visa plan-output som text

```bash
# Använd terraform show lokalt efter export
dagger call new --source . plan export --path ./output.tfplan
terraform show output.tfplan

# Eller använd Plan med no-color för direkt output
dagger call new --source . plan --extra-args="-no-color"
```

### Validera konfiguration

```bash
dagger call new --source . validate
```

### Komplett workflow

```bash
# 1. Skapa plan
dagger call new --source . plan export --path ./output.tfplan

# 2. Granska planen (manuellt)
terraform show output.tfplan

# 3. Applicera planen
dagger call new --source . apply --plan-file ./output.tfplan
```

## Funktioner

- **New**: Skapar en bas-container med Terraform-miljön
- **Init**: Initialiserar Terraform working directory (stödjer extraArgs)
- **Plan**: Skapar en execution plan och sparar som output.tfplan fil (kör Init automatiskt, stödjer extraArgs)
- **Apply**: Applicerar en specifik plan-fil med auto-approve (kör Init automatiskt, stödjer extraArgs)
- **Validate**: Validerar Terraform-konfigurationen (kör Init automatiskt)

## Extra Arguments

Funktionerna Init, Plan och Apply stödjer extra argument för flexibilitet:

```bash
# Plan med target
dagger call new --source . plan --extra-args="-target=aws_instance.web"

# Init med backend config
dagger call new --source . init --extra-args="-backend-config=bucket=my-state"

# Apply med parallelism
dagger call new --source . apply --plan-file ./plan.tfplan --extra-args="-parallelism=10"
```

## Plan-baserat Workflow

Modulen följer Terraform best practices med plan-filer:

1. **Plan**: Skapar en plan-fil (`output.tfplan`) som kan granskas
2. **Apply**: Applicerar en specifik plan-fil med auto-approve

```bash
# Säkert workflow med plan-fil
dagger call new --source . plan export --path ./output.tfplan
dagger call new --source . apply --plan-file ./output.tfplan
```

## Krav

- Dagger CLI installerat
- Terraform-filer i din workspace