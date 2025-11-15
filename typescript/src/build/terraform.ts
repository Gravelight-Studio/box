import * as fs from 'fs';
import * as path from 'path';
import { Handler, DeploymentType } from '../annotations';
import { Logger } from './types';

/**
 * Generator for Terraform infrastructure as code
 */
export class TerraformGenerator {
  constructor(
    private handlers: Handler[],
    private outputDir: string,
    private moduleName: string,
    private projectId: string,
    private region: string,
    private logger: Logger
  ) {}

  /**
   * Generate Terraform configuration
   */
  async generate(): Promise<void> {
    this.logger.info('Generating Terraform configuration', {
      handlers: this.handlers.length,
      outputDir: this.outputDir
    });

    // Create output directory
    await fs.promises.mkdir(this.outputDir, { recursive: true });

    // Generate main configuration
    await this.generateMain();

    // Generate variables
    await this.generateVariables();

    // Generate outputs
    await this.generateOutputs();

    // Generate functions module
    await this.generateFunctionsModule();

    // Generate containers module
    await this.generateContainersModule();

    this.logger.info('Generated Terraform configuration', {
      terraformDir: this.outputDir
    });
  }

  /**
   * Generate main.tf
   */
  private async generateMain(): Promise<void> {
    const functionHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Function);
    const containerHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Container);

    const tf = `# Main Terraform configuration
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

# Enable required APIs
resource "google_project_service" "cloud_functions" {
  count   = ${functionHandlers.length > 0 ? 1 : 0}
  service = "cloudfunctions.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloud_run" {
  count   = ${containerHandlers.length > 0 ? 1 : 0}
  service = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloud_build" {
  service = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

# Cloud Functions
${functionHandlers.length > 0 ? 'module "functions" {\n  source = "./modules/functions"\n  \n  project_id = var.project_id\n  region     = var.region\n  environment = var.environment\n  \n  depends_on = [google_project_service.cloud_functions]\n}' : ''}

# Cloud Run Containers
${containerHandlers.length > 0 ? 'module "containers" {\n  source = "./modules/containers"\n  \n  project_id = var.project_id\n  region     = var.region\n  environment = var.environment\n  \n  depends_on = [google_project_service.cloud_run]\n}' : ''}
`;

    await fs.promises.writeFile(path.join(this.outputDir, 'main.tf'), tf);
  }

  /**
   * Generate variables.tf
   */
  private async generateVariables(): Promise<void> {
    const tf = `# Terraform variables
variable "project_id" {
  description = "GCP project ID"
  type        = string
  default     = "${this.projectId}"
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "${this.region}"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}
`;

    await fs.promises.writeFile(path.join(this.outputDir, 'variables.tf'), tf);
  }

  /**
   * Generate outputs.tf
   */
  private async generateOutputs(): Promise<void> {
    const functionHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Function);
    const containerHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Container);

    const tf = `# Terraform outputs
${functionHandlers.length > 0 ? 'output "function_urls" {\n  description = "URLs of deployed Cloud Functions"\n  value       = module.functions.function_urls\n}' : ''}

${containerHandlers.length > 0 ? 'output "container_urls" {\n  description = "URLs of deployed Cloud Run services"\n  value       = module.containers.service_urls\n}' : ''}
`;

    await fs.promises.writeFile(path.join(this.outputDir, 'outputs.tf'), tf);
  }

  /**
   * Generate functions module
   */
  private async generateFunctionsModule(): Promise<void> {
    const functionHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Function);

    if (functionHandlers.length === 0) {
      return;
    }

    const moduleDir = path.join(this.outputDir, 'modules', 'functions');
    await fs.promises.mkdir(moduleDir, { recursive: true });

    // Generate module main.tf
    const tf = `# Cloud Functions module
${functionHandlers.map(h => this.generateFunctionResource(h)).join('\n\n')}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'main.tf'), tf);

    // Generate module variables.tf
    const variables = `variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "environment" {
  type = string
}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'variables.tf'), variables);

    // Generate module outputs.tf
    const outputs = `output "function_urls" {
  value = {
${functionHandlers.map(h => `    "${this.getFunctionName(h)}" = google_cloudfunctions2_function.${this.getFunctionName(h).replace(/-/g, '_')}.url`).join('\n')}
  }
}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'outputs.tf'), outputs);
  }

  /**
   * Generate Terraform resource for a function
   */
  private generateFunctionResource(handler: Handler): string {
    const functionName = this.getFunctionName(handler);
    const resourceName = functionName.replace(/-/g, '_');

    return `resource "google_cloudfunctions2_function" "${resourceName}" {
  name        = "${functionName}"
  location    = var.region
  description = "${handler.route.method} ${handler.route.path}"

  build_config {
    runtime     = "nodejs20"
    entry_point = "${handler.functionName}"
    source {
      storage_source {
        bucket = google_storage_bucket.functions_bucket.name
        object = google_storage_bucket_object.${resourceName}_source.name
      }
    }
  }

  service_config {
    max_instance_count = 100
    min_instance_count = 0
    available_memory   = "${handler.memory?.size || 256}M"
    timeout_seconds    = ${handler.timeout ? Math.floor(handler.timeout.duration / 1000) : 60}

    environment_variables = {
      NODE_ENV      = "production"
      FUNCTION_NAME = "${handler.functionName}"
    }
  }
}

resource "google_storage_bucket_object" "${resourceName}_source" {
  name   = "${functionName}-\${var.environment}-source.zip"
  bucket = google_storage_bucket.functions_bucket.name
  source = "../functions/${functionName}/function.zip"
}

resource "google_cloudfunctions2_function_iam_member" "${resourceName}_invoker" {
  project        = var.project_id
  location       = var.region
  cloud_function = google_cloudfunctions2_function.${resourceName}.name
  role           = "roles/cloudfunctions.invoker"
  member         = "allUsers"
}`;
  }

  /**
   * Generate containers module
   */
  private async generateContainersModule(): Promise<void> {
    const containerHandlers = this.handlers.filter(h => h.deploymentType === DeploymentType.Container);

    if (containerHandlers.length === 0) {
      return;
    }

    const moduleDir = path.join(this.outputDir, 'modules', 'containers');
    await fs.promises.mkdir(moduleDir, { recursive: true });

    // Group by service
    const services = new Set(containerHandlers.map(h => h.packageName));

    // Generate module main.tf
    const tf = `# Cloud Run module
${Array.from(services).map(s => this.generateContainerService(s)).join('\n\n')}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'main.tf'), tf);

    // Generate module variables.tf
    const variables = `variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "environment" {
  type = string
}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'variables.tf'), variables);

    // Generate module outputs.tf
    const outputs = `output "service_urls" {
  value = {
${Array.from(services).map(s => `    "${s}" = google_cloud_run_service.${s}.status[0].url`).join('\n')}
  }
}
`;

    await fs.promises.writeFile(path.join(moduleDir, 'outputs.tf'), outputs);
  }

  /**
   * Generate Cloud Run service resource
   */
  private generateContainerService(serviceName: string): string {
    return `resource "google_cloud_run_service" "${serviceName}" {
  name     = "${serviceName}"
  location = var.region

  template {
    spec {
      containers {
        image = "gcr.io/\${var.project_id}/${serviceName}:latest"

        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }

        env {
          name  = "NODE_ENV"
          value = "production"
        }
      }

      container_concurrency = 100
      timeout_seconds       = 300
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "100"
        "autoscaling.knative.dev/minScale" = "0"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

resource "google_cloud_run_service_iam_member" "${serviceName}_invoker" {
  service  = google_cloud_run_service.${serviceName}.name
  location = google_cloud_run_service.${serviceName}.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}`;
  }

  /**
   * Get normalized function name
   */
  private getFunctionName(handler: Handler): string {
    return handler.functionName
      .replace(/([A-Z])/g, '-$1')
      .toLowerCase()
      .replace(/^-/, '');
  }
}
