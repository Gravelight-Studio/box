package build

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// TerraformGenerator generates Terraform Infrastructure as Code
type TerraformGenerator struct {
	handlers    []annotations.Handler
	outputDir   string
	moduleName  string
	projectID   string
	region      string
	environment string // dev, staging, production
	logger      *zap.Logger
}

// Generate creates complete Terraform configuration
func (tg *TerraformGenerator) Generate() error {
	if len(tg.handlers) == 0 {
		tg.logger.Info("No handlers to generate Terraform configuration")
		return nil
	}

	// Create terraform output directory
	if err := os.MkdirAll(tg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create terraform directory: %w", err)
	}

	tg.logger.Info("Generating Terraform configuration",
		zap.Int("handlers", len(tg.handlers)),
		zap.String("output_dir", tg.outputDir))

	// Generate module directories
	if err := tg.generateModuleStructure(); err != nil {
		return fmt.Errorf("failed to generate module structure: %w", err)
	}

	// Generate cloud-functions module
	if err := tg.generateCloudFunctionsModule(); err != nil {
		return fmt.Errorf("failed to generate cloud-functions module: %w", err)
	}

	// Generate cloud-run module
	if err := tg.generateCloudRunModule(); err != nil {
		return fmt.Errorf("failed to generate cloud-run module: %w", err)
	}

	// Generate api-gateway module
	if err := tg.generateAPIGatewayModule(); err != nil {
		return fmt.Errorf("failed to generate api-gateway module: %w", err)
	}

	// Generate networking module
	if err := tg.generateNetworkingModule(); err != nil {
		return fmt.Errorf("failed to generate networking module: %w", err)
	}

	// Generate root configuration files
	if err := tg.generateRootMain(); err != nil {
		return fmt.Errorf("failed to generate root main.tf: %w", err)
	}

	if err := tg.generateVariables(); err != nil {
		return fmt.Errorf("failed to generate variables.tf: %w", err)
	}

	if err := tg.generateOutputs(); err != nil {
		return fmt.Errorf("failed to generate outputs.tf: %w", err)
	}

	// Generate environment-specific tfvars
	if err := tg.generateEnvironmentFiles(); err != nil {
		return fmt.Errorf("failed to generate environment files: %w", err)
	}

	// Generate supporting files
	if err := tg.generateGitignore(); err != nil {
		return fmt.Errorf("failed to generate .gitignore: %w", err)
	}

	if err := tg.generateREADME(); err != nil {
		return fmt.Errorf("failed to generate README.md: %w", err)
	}

	tg.logger.Info("Generated Terraform configuration",
		zap.String("terraform_dir", tg.outputDir))

	return nil
}

// generateModuleStructure creates the module directory structure
func (tg *TerraformGenerator) generateModuleStructure() error {
	modules := []string{
		"modules/cloud-functions",
		"modules/cloud-run",
		"modules/api-gateway",
		"modules/networking",
		"environments",
	}

	for _, module := range modules {
		modulePath := filepath.Join(tg.outputDir, module)
		if err := os.MkdirAll(modulePath, 0755); err != nil {
			return fmt.Errorf("failed to create module directory %s: %w", module, err)
		}
	}

	return nil
}

// generateCloudFunctionsModule generates the cloud-functions module
func (tg *TerraformGenerator) generateCloudFunctionsModule() error {
	modulePath := filepath.Join(tg.outputDir, "modules", "cloud-functions")

	// Filter function handlers
	functions := filterFunctionHandlers(tg.handlers)
	if len(functions) == 0 {
		tg.logger.Info("No cloud functions to generate in Terraform")
		return nil
	}

	// Get unique service accounts (by package)
	serviceAccounts := tg.getServiceAccounts(functions)

	// Generate main.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "main.tf"),
		cloudFunctionsMainTemplate,
		map[string]interface{}{
			"Functions":       functions,
			"ServiceAccounts": serviceAccounts,
		},
	); err != nil {
		return err
	}

	// Generate variables.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "variables.tf"),
		cloudFunctionsVariablesTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate outputs.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "outputs.tf"),
		cloudFunctionsOutputsTemplate,
		map[string]interface{}{
			"Functions": functions,
		},
	); err != nil {
		return err
	}

	tg.logger.Info("Generated cloud-functions module",
		zap.Int("functions", len(functions)),
		zap.Int("service_accounts", len(serviceAccounts)))

	return nil
}

// generateCloudRunModule generates the cloud-run module
func (tg *TerraformGenerator) generateCloudRunModule() error {
	modulePath := filepath.Join(tg.outputDir, "modules", "cloud-run")

	// Filter container handlers and group by service
	containers := filterContainerHandlers(tg.handlers)
	if len(containers) == 0 {
		tg.logger.Info("No cloud run services to generate in Terraform")
		return nil
	}

	serviceGroups := tg.groupHandlersByPackage(containers)
	serviceAccounts := tg.getServiceAccounts(containers)

	// Generate main.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "main.tf"),
		cloudRunMainTemplate,
		map[string]interface{}{
			"ServiceGroups":   serviceGroups,
			"ServiceAccounts": serviceAccounts,
		},
	); err != nil {
		return err
	}

	// Generate variables.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "variables.tf"),
		cloudRunVariablesTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate outputs.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "outputs.tf"),
		cloudRunOutputsTemplate,
		map[string]interface{}{
			"ServiceGroups": serviceGroups,
		},
	); err != nil {
		return err
	}

	tg.logger.Info("Generated cloud-run module",
		zap.Int("services", len(serviceGroups)),
		zap.Int("service_accounts", len(serviceAccounts)))

	return nil
}

// generateAPIGatewayModule generates the api-gateway module
func (tg *TerraformGenerator) generateAPIGatewayModule() error {
	modulePath := filepath.Join(tg.outputDir, "modules", "api-gateway")

	if len(tg.handlers) == 0 {
		return nil
	}

	// Generate main.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "main.tf"),
		apiGatewayMainTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate variables.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "variables.tf"),
		apiGatewayVariablesTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate outputs.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "outputs.tf"),
		apiGatewayOutputsTemplate,
		nil,
	); err != nil {
		return err
	}

	tg.logger.Info("Generated api-gateway module")

	return nil
}

// generateNetworkingModule generates the networking module
func (tg *TerraformGenerator) generateNetworkingModule() error {
	modulePath := filepath.Join(tg.outputDir, "modules", "networking")

	// Generate main.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "main.tf"),
		networkingMainTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate variables.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "variables.tf"),
		networkingVariablesTemplate,
		nil,
	); err != nil {
		return err
	}

	// Generate outputs.tf
	if err := tg.generateFile(
		filepath.Join(modulePath, "outputs.tf"),
		networkingOutputsTemplate,
		nil,
	); err != nil {
		return err
	}

	tg.logger.Info("Generated networking module")

	return nil
}

// generateRootMain generates the root main.tf file
func (tg *TerraformGenerator) generateRootMain() error {
	hasFunctions := len(filterFunctionHandlers(tg.handlers)) > 0
	hasContainers := len(filterContainerHandlers(tg.handlers)) > 0

	return tg.generateFile(
		filepath.Join(tg.outputDir, "main.tf"),
		rootMainTemplate,
		map[string]interface{}{
			"HasFunctions":  hasFunctions,
			"HasContainers": hasContainers,
		},
	)
}

// generateVariables generates the variables.tf file
func (tg *TerraformGenerator) generateVariables() error {
	return tg.generateFile(
		filepath.Join(tg.outputDir, "variables.tf"),
		rootVariablesTemplate,
		nil,
	)
}

// generateOutputs generates the outputs.tf file
func (tg *TerraformGenerator) generateOutputs() error {
	hasFunctions := len(filterFunctionHandlers(tg.handlers)) > 0
	hasContainers := len(filterContainerHandlers(tg.handlers)) > 0

	return tg.generateFile(
		filepath.Join(tg.outputDir, "outputs.tf"),
		rootOutputsTemplate,
		map[string]interface{}{
			"HasFunctions":  hasFunctions,
			"HasContainers": hasContainers,
		},
	)
}

// generateEnvironmentFiles generates environment-specific tfvars files
func (tg *TerraformGenerator) generateEnvironmentFiles() error {
	environments := []string{"dev", "staging", "production"}

	for _, env := range environments {
		if err := tg.generateFile(
			filepath.Join(tg.outputDir, "environments", fmt.Sprintf("%s.tfvars", env)),
			environmentTfvarsTemplate,
			map[string]interface{}{
				"Environment": env,
			},
		); err != nil {
			return err
		}
	}

	tg.logger.Info("Generated environment tfvars files", zap.Int("count", len(environments)))

	return nil
}

// generateGitignore generates .gitignore for Terraform
func (tg *TerraformGenerator) generateGitignore() error {
	return tg.generateFile(
		filepath.Join(tg.outputDir, ".gitignore"),
		terraformGitignoreTemplate,
		nil,
	)
}

// generateREADME generates README.md with usage instructions
func (tg *TerraformGenerator) generateREADME() error {
	hasFunctions := len(filterFunctionHandlers(tg.handlers)) > 0
	hasContainers := len(filterContainerHandlers(tg.handlers)) > 0

	return tg.generateFile(
		filepath.Join(tg.outputDir, "README.md"),
		terraformREADMETemplate,
		map[string]interface{}{
			"HasFunctions":  hasFunctions,
			"HasContainers": hasContainers,
		},
	)
}

// Helper functions

// generateFile creates a file from a template
func (tg *TerraformGenerator) generateFile(path string, templateStr string, data interface{}) error {
	tmpl := template.Must(template.New("terraform").Funcs(template.FuncMap{
		"toKebabCase": toKebabCase,
		"toSnakeCase": toSnakeCase,
		"replace":     strings.ReplaceAll,
		"toUpper":     strings.ToUpper,
		"toLower":     strings.ToLower,
		"stripMB": func(s string) string {
			return strings.TrimSuffix(s, "MB")
		},
	}).Parse(templateStr))

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

// getServiceAccounts gets unique service accounts from handlers (by package)
func (tg *TerraformGenerator) getServiceAccounts(handlers []annotations.Handler) []string {
	saMap := make(map[string]bool)
	for _, h := range handlers {
		if h.PackageName != "" {
			saMap[h.PackageName] = true
		}
	}

	sas := make([]string, 0, len(saMap))
	for sa := range saMap {
		sas = append(sas, sa)
	}
	sort.Strings(sas)
	return sas
}

// groupHandlersByPackage groups handlers by package name
func (tg *TerraformGenerator) groupHandlersByPackage(handlers []annotations.Handler) []ServiceGroup {
	groupMap := make(map[string][]annotations.Handler)

	for _, handler := range handlers {
		packageName := handler.PackageName
		if packageName == "" {
			packageName = "default"
		}
		groupMap[packageName] = append(groupMap[packageName], handler)
	}

	groups := make([]ServiceGroup, 0, len(groupMap))
	for name, handlers := range groupMap {
		groups = append(groups, ServiceGroup{
			Name:     name,
			Handlers: handlers,
		})
	}

	// Sort for consistent output
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups
}

// toSnakeCase converts "CreateAccount" to "create_account"
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// Templates

const cloudFunctionsMainTemplate = `# Cloud Functions Module
# Generated by Wylla build system

{{range .ServiceAccounts}}
# Service account for {{.}} service
resource "google_service_account" "{{. | toSnakeCase}}" {
  account_id   = "wylla-{{.}}-$${var.environment}"
  display_name = "Wylla {{.}} Service ($${var.environment})"
  description  = "Service account for {{.}} handlers"
}

# Grant Cloud SQL client role
resource "google_project_iam_member" "{{. | toSnakeCase}}_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:$${google_service_account.{{. | toSnakeCase}}.email}"
}

# Grant Secret Manager accessor role
resource "google_project_iam_member" "{{. | toSnakeCase}}_secrets" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:$${google_service_account.{{. | toSnakeCase}}.email}"
}
{{end}}

# Storage bucket for function source code
resource "google_storage_bucket" "functions" {
  name          = "$${var.project_id}-functions-$${var.environment}"
  location      = var.region
  force_destroy = var.environment != "production"

  uniform_bucket_level_access = true
}

{{range .Functions}}
# Function: {{.FunctionName}}
resource "google_cloudfunctions_function" "{{.FunctionName | toSnakeCase}}" {
  name                  = "wylla-$${var.environment}-{{.FunctionName | toKebabCase}}"
  description           = "{{.FunctionName}} handler"
  runtime              = "go122"
  entry_point          = "{{.FunctionName}}"
  service_account_email = google_service_account.{{.PackageName | toSnakeCase}}.email

  available_memory_mb = {{if .Memory}}{{.Memory | stripMB}}{{else}}256{{end}}
  timeout             = {{if .Timeout}}{{.Timeout.Seconds | printf "%.0f"}}{{else}}60{{end}}

  source_archive_bucket = google_storage_bucket.functions.name
  source_archive_object = "{{.FunctionName | toKebabCase}}.zip"

  trigger_http = true

  environment_variables = {
    DATABASE_URL = data.google_secret_manager_secret_version.database_url.secret_data
    ENVIRONMENT  = var.environment
  }

  depends_on = [
    google_project_iam_member.{{.PackageName | toSnakeCase}}_cloudsql,
    google_project_iam_member.{{.PackageName | toSnakeCase}}_secrets
  ]
}

# Allow unauthenticated access (API Gateway will handle auth)
resource "google_cloudfunctions_function_iam_member" "{{.FunctionName | toSnakeCase}}_invoker" {
  cloud_function = google_cloudfunctions_function.{{.FunctionName | toSnakeCase}}.name
  role           = "roles/cloudfunctions.invoker"
  member         = "allUsers"
}
{{end}}

# Reference to database URL secret
data "google_secret_manager_secret_version" "database_url" {
  secret  = "database-url-$${var.environment}"
  version = "latest"
}
`

const cloudFunctionsVariablesTemplate = `# Cloud Functions Module Variables

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}
`

const cloudFunctionsOutputsTemplate = `# Cloud Functions Module Outputs

{{range .Functions}}
output "{{.FunctionName | toSnakeCase}}_url" {
  description = "URL for {{.FunctionName}} function"
  value       = google_cloudfunctions_function.{{.FunctionName | toSnakeCase}}.https_trigger_url
}
{{end}}

output "function_urls" {
  description = "Map of all function URLs"
  value = {
{{range .Functions}}    "{{.FunctionName}}" = google_cloudfunctions_function.{{.FunctionName | toSnakeCase}}.https_trigger_url
{{end}}  }
}
`

const cloudRunMainTemplate = `# Cloud Run Module
# Generated by Wylla build system

{{range .ServiceAccounts}}
# Service account for {{.}} service
resource "google_service_account" "{{. | toSnakeCase}}" {
  account_id   = "wylla-{{.}}-$${var.environment}"
  display_name = "Wylla {{.}} Service ($${var.environment})"
  description  = "Service account for {{.}} service"
}

# Grant Cloud SQL client role
resource "google_project_iam_member" "{{. | toSnakeCase}}_cloudsql" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:$${google_service_account.{{. | toSnakeCase}}.email}"
}

# Grant Secret Manager accessor role
resource "google_project_iam_member" "{{. | toSnakeCase}}_secrets" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:$${google_service_account.{{. | toSnakeCase}}.email}"
}
{{end}}

{{range .ServiceGroups}}
# Cloud Run Service: {{.Name}}
resource "google_cloud_run_service" "{{.Name | toSnakeCase}}" {
  name     = "wylla-$${var.environment}-{{.Name}}"
  location = var.region

  template {
    spec {
      service_account_name = google_service_account.{{.Name | toSnakeCase}}.email

      containers {
        image = "gcr.io/$${var.project_id}/{{.Name}}:latest"

        ports {
          container_port = 8080
        }

        env {
          name  = "DATABASE_URL"
          value = data.google_secret_manager_secret_version.database_url.secret_data
        }

        env {
          name  = "ENVIRONMENT"
          value = var.environment
        }

        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }
      }

      container_concurrency = 80
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "10"
        "run.googleapis.com/client-name"   = "terraform"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  depends_on = [
    google_project_iam_member.{{.Name | toSnakeCase}}_cloudsql,
    google_project_iam_member.{{.Name | toSnakeCase}}_secrets
  ]
}

# Allow unauthenticated access (API Gateway will handle auth)
resource "google_cloud_run_service_iam_member" "{{.Name | toSnakeCase}}_invoker" {
  service  = google_cloud_run_service.{{.Name | toSnakeCase}}.name
  location = google_cloud_run_service.{{.Name | toSnakeCase}}.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
{{end}}

# Reference to database URL secret
data "google_secret_manager_secret_version" "database_url" {
  secret  = "database-url-$${var.environment}"
  version = "latest"
}
`

const cloudRunVariablesTemplate = `# Cloud Run Module Variables

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}
`

const cloudRunOutputsTemplate = `# Cloud Run Module Outputs

{{range .ServiceGroups}}
output "{{.Name | toSnakeCase}}_url" {
  description = "URL for {{.Name}} service"
  value       = google_cloud_run_service.{{.Name | toSnakeCase}}.status[0].url
}
{{end}}

output "service_urls" {
  description = "Map of all service URLs"
  value = {
{{range .ServiceGroups}}    "{{.Name}}" = google_cloud_run_service.{{.Name | toSnakeCase}}.status[0].url
{{end}}  }
}
`

const apiGatewayMainTemplate = `# API Gateway Module
# Generated by Wylla build system

# API Gateway API
resource "google_api_gateway_api" "api" {
  api_id       = "wylla-api-$${var.environment}"
  display_name = "Wylla API ($${var.environment})"
}

# API Gateway API Config
resource "google_api_gateway_api_config" "api_config" {
  api           = google_api_gateway_api.api.api_id
  api_config_id = "wylla-api-config-$${var.environment}-$${formatdate("YYYYMMDDhhmmss", timestamp())}"
  display_name  = "Wylla API Config ($${var.environment})"

  openapi_documents {
    document {
      path     = "openapi.yaml"
      contents = filebase64("$${path.module}/../../gateway/openapi.yaml")
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

# API Gateway Gateway
resource "google_api_gateway_gateway" "gateway" {
  api_config   = google_api_gateway_api_config.api_config.id
  gateway_id   = "wylla-gateway-$${var.environment}"
  display_name = "Wylla API Gateway ($${var.environment})"
  region       = var.region
}
`

const apiGatewayVariablesTemplate = `# API Gateway Module Variables

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}
`

const apiGatewayOutputsTemplate = `# API Gateway Module Outputs

output "gateway_url" {
  description = "API Gateway URL"
  value       = google_api_gateway_gateway.gateway.default_hostname
}

output "api_id" {
  description = "API Gateway API ID"
  value       = google_api_gateway_api.api.api_id
}

output "gateway_id" {
  description = "API Gateway Gateway ID"
  value       = google_api_gateway_gateway.gateway.gateway_id
}
`

const networkingMainTemplate = `# Networking Module
# Generated by Wylla build system

# VPC Network
resource "google_compute_network" "vpc" {
  name                    = "wylla-vpc-$${var.environment}"
  auto_create_subnetworks = false
  description             = "Wylla VPC network for $${var.environment}"
}

# Subnet
resource "google_compute_subnetwork" "subnet" {
  name          = "wylla-subnet-$${var.environment}"
  ip_cidr_range = "10.0.0.0/24"
  region        = var.region
  network       = google_compute_network.vpc.id
}

# VPC Access Connector (for functions/Cloud Run to access VPC)
resource "google_vpc_access_connector" "connector" {
  name          = "wylla-connector-$${var.environment}"
  region        = var.region
  ip_cidr_range = "10.8.0.0/28"
  network       = google_compute_network.vpc.name
}

# Cloud SQL Instance
resource "google_sql_database_instance" "main" {
  name             = "wylla-db-$${var.environment}"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier = var.db_tier

    ip_configuration {
      ipv4_enabled                                  = true
      private_network                               = google_compute_network.vpc.id
      enable_private_path_for_google_cloud_services = true
    }

    backup_configuration {
      enabled            = var.environment == "production"
      start_time         = "03:00"
      point_in_time_recovery_enabled = var.environment == "production"
    }

    database_flags {
      name  = "max_connections"
      value = var.max_connections
    }
  }

  deletion_protection = var.environment == "production"
}

# Database
resource "google_sql_database" "database" {
  name     = var.database_name
  instance = google_sql_database_instance.main.name
}

# Database user
resource "google_sql_user" "user" {
  name     = var.database_user
  instance = google_sql_database_instance.main.name
  password = var.database_password
}
`

const networkingVariablesTemplate = `# Networking Module Variables

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}

variable "db_tier" {
  description = "Cloud SQL instance tier"
  type        = string
  default     = "db-f1-micro"
}

variable "max_connections" {
  description = "Maximum database connections"
  type        = string
  default     = "100"
}

variable "database_name" {
  description = "Database name"
  type        = string
  default     = "wylla"
}

variable "database_user" {
  description = "Database user"
  type        = string
  default     = "wylla"
}

variable "database_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}
`

const networkingOutputsTemplate = `# Networking Module Outputs

output "vpc_id" {
  description = "VPC network ID"
  value       = google_compute_network.vpc.id
}

output "vpc_name" {
  description = "VPC network name"
  value       = google_compute_network.vpc.name
}

output "subnet_id" {
  description = "Subnet ID"
  value       = google_compute_subnetwork.subnet.id
}

output "connector_id" {
  description = "VPC Access Connector ID"
  value       = google_vpc_access_connector.connector.id
}

output "database_instance" {
  description = "Cloud SQL instance name"
  value       = google_sql_database_instance.main.name
}

output "database_connection_name" {
  description = "Cloud SQL connection name"
  value       = google_sql_database_instance.main.connection_name
}

output "database_private_ip" {
  description = "Cloud SQL private IP address"
  value       = google_sql_database_instance.main.private_ip_address
}
`

const rootMainTemplate = `# Wylla Backend Infrastructure
# Generated by Wylla build system
# DO NOT EDIT - This file is auto-generated

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

{{if .HasFunctions}}
# Cloud Functions Module
module "cloud_functions" {
  source = "./modules/cloud-functions"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment
}
{{end}}

{{if .HasContainers}}
# Cloud Run Module
module "cloud_run" {
  source = "./modules/cloud-run"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment
}
{{end}}

# API Gateway Module
module "api_gateway" {
  source = "./modules/api-gateway"

  project_id  = var.project_id
  region      = var.region
  environment = var.environment
}

# Networking Module
module "networking" {
  source = "./modules/networking"

  project_id        = var.project_id
  region            = var.region
  environment       = var.environment
  database_password = var.database_password
}
`

const rootVariablesTemplate = `# Root Module Variables

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "Environment must be dev, staging, or production."
  }
}

variable "database_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}
`

const rootOutputsTemplate = `# Root Module Outputs

{{if .HasFunctions}}
output "function_urls" {
  description = "Cloud Function URLs"
  value       = module.cloud_functions.function_urls
}
{{end}}

{{if .HasContainers}}
output "service_urls" {
  description = "Cloud Run service URLs"
  value       = module.cloud_run.service_urls
}
{{end}}

output "api_gateway_url" {
  description = "API Gateway URL"
  value       = module.api_gateway.gateway_url
}

output "database_connection_name" {
  description = "Cloud SQL connection name"
  value       = module.networking.database_connection_name
}
`

const environmentTfvarsTemplate = `# Terraform variables for {{.Environment}} environment
# Generated by Wylla build system

project_id  = "YOUR_PROJECT_ID"
region      = "us-central1"
environment = "{{.Environment}}"

# Database password - CHANGE THIS!
# Better: Store in Secret Manager and reference via data source
database_password = "CHANGE_ME_{{.Environment | toUpper}}"
`

const terraformGitignoreTemplate = `# Terraform
*.tfstate
*.tfstate.backup
*.tfstate.lock.info
.terraform/
.terraform.lock.hcl

# Sensitive files
terraform.tfvars
*.auto.tfvars
*_override.tf
*_override.tf.json

# Crash logs
crash.log
crash.*.log

# CLI config
.terraformrc
terraform.rc
`

const terraformREADMETemplate = `# Wylla Backend Terraform Configuration

Generated by Wylla build system.

## Overview

This Terraform configuration manages the complete infrastructure for the Wylla backend:

{{if .HasFunctions}}- **Cloud Functions**: Serverless functions for individual handlers
{{end}}{{if .HasContainers}}- **Cloud Run**: Container services for grouped handlers
{{end}}- **API Gateway**: Unified API Gateway with OpenAPI 3.0 spec
- **Networking**: VPC, Cloud SQL database, VPC connectors

## Prerequisites

1. **Terraform**: Install Terraform >= 1.0
2. **GCP Account**: Active GCP project with billing enabled
3. **gcloud CLI**: Authenticated with ` + "`" + `gcloud auth application-default login` + "`" + `
4. **Secrets**: Create required secrets in Secret Manager

### Create Secrets

Before applying Terraform, create the required secrets:

` + "```bash" + `
# Database URL secret
echo -n "postgres://user:pass@host/dbname" | \
  gcloud secrets create database-url-dev --data-file=-

echo -n "postgres://user:pass@host/dbname" | \
  gcloud secrets create database-url-staging --data-file=-

echo -n "postgres://user:pass@host/dbname" | \
  gcloud secrets create database-url-production --data-file=-
` + "```" + `

## Usage

### 1. Configure Environment

Edit the appropriate environment file:

` + "```bash" + `
# For dev environment
vim environments/dev.tfvars

# Update:
# - project_id: Your GCP project ID
# - region: Your preferred region
# - database_password: Secure password
` + "```" + `

### 2. Initialize Terraform

` + "```bash" + `
terraform init
` + "```" + `

### 3. Plan Deployment

` + "```bash" + `
# Review changes for dev environment
terraform plan -var-file=environments/dev.tfvars

# Or for staging
terraform plan -var-file=environments/staging.tfvars

# Or for production
terraform plan -var-file=environments/production.tfvars
` + "```" + `

### 4. Apply Infrastructure

` + "```bash" + `
# Deploy to dev
terraform apply -var-file=environments/dev.tfvars

# Deploy to staging
terraform apply -var-file=environments/staging.tfvars

# Deploy to production (be careful!)
terraform apply -var-file=environments/production.tfvars
` + "```" + `

### 5. View Outputs

` + "```bash" + `
terraform output

# Get API Gateway URL
terraform output api_gateway_url
` + "```" + `

## Module Structure

` + "```" + `
.
├── main.tf                    # Root configuration
├── variables.tf               # Input variables
├── outputs.tf                 # Output values
├── .gitignore                 # Ignore sensitive files
├── README.md                  # This file
├── modules/
│   ├── cloud-functions/       # Cloud Functions module
│   ├── cloud-run/            # Cloud Run module
│   ├── api-gateway/          # API Gateway module
│   └── networking/           # VPC, Cloud SQL module
└── environments/
    ├── dev.tfvars            # Dev environment config
    ├── staging.tfvars        # Staging environment config
    └── production.tfvars     # Production environment config
` + "```" + `

## Important Notes

### State Management

This configuration uses **local state**. State files are stored locally in ` + "`" + `terraform.tfstate` + "`" + `.

⚠️ **Important**:
- Never commit state files to git (already in .gitignore)
- State files may contain sensitive data
- Keep backups of state files
- Consider migrating to remote state (GCS) for team collaboration

### Security

1. **Secrets**: Use Secret Manager for all sensitive data
2. **Service Accounts**: Each service has its own service account with least-privilege IAM
3. **Passwords**: Never commit passwords. Use environment variables or Secret Manager
4. **tfvars**: Never commit ` + "`" + `*.tfvars` + "`" + ` files with real credentials

### Costs

Estimated monthly costs (varies by usage):
- Cloud Functions: Pay per invocation (~$0.40/million requests)
- Cloud Run: Pay per use (~$0.00002400/vCPU-second)
- Cloud SQL: $7-50+/month depending on tier
- API Gateway: $3/million requests
- VPC: Minimal (~$5/month)

Use ` + "`" + `terraform plan` + "`" + ` to estimate before applying.

## Troubleshooting

### Authentication Errors

` + "```bash" + `
# Re-authenticate
gcloud auth application-default login
` + "```" + `

### API Not Enabled

Enable required APIs:

` + "```bash" + `
gcloud services enable cloudfunctions.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable apigateway.googleapis.com
gcloud services enable vpcaccess.googleapis.com
gcloud services enable sqladmin.googleapis.com
` + "```" + `

### State Locked

If state is locked after failed apply:

` + "```bash" + `
terraform force-unlock <LOCK_ID>
` + "```" + `

## Cleanup

To destroy all infrastructure:

` + "```bash" + `
# DANGER: This deletes everything
terraform destroy -var-file=environments/dev.tfvars
` + "```" + `

## Next Steps

After infrastructure is deployed:

1. **Upload Function Code**: Deploy function packages to Cloud Functions
2. **Build Container Images**: Build and push containers to GCR
3. **Configure DNS**: Point custom domain to API Gateway
4. **Monitor**: Set up Cloud Monitoring and Logging
5. **CI/CD**: Integrate with Cloud Build or GitHub Actions

## Support

For issues or questions:
- Review Terraform docs: https://www.terraform.io/docs
- GCP Terraform provider: https://registry.terraform.io/providers/hashicorp/google/latest/docs
- Wylla documentation: See project README

---

Generated by Wylla Framework
`
