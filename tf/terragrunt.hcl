# vim: set syntax=terraform:

terraform_version_constraint  = "~> 1.5"
terragrunt_version_constraint = ">= 0.45"

locals {
  user         = get_env("USER", "NONE")
  stack        = get_env("STACK", "dev")
  stack_file   = "${get_parent_terragrunt_dir()}/stacks/${local.stack}.hcl"
  stack_config = read_terragrunt_config(local.stack_file)
  s3_prefix    = "emprovise"
  s3_suffix    = "${local.stack}-${local.stack_config.inputs.region}"
  default_tags = {
    Stage       = local.stack_config.inputs.stage
    CreatedBy   = local.user
    Environment = "EMPROVISE"
  }
}

remote_state {
  backend = "s3"
  config = {
    bucket         = "${local.s3_prefix}-terraform-${local.s3_suffix}"
    key            = "${path_relative_to_include()}/terraform.tfstate"
    region         = local.stack_config.inputs.region
    encrypt        = true
    dynamodb_table = "terraform-state-locking"
    s3_bucket_tags = local.default_tags
  }
  disable_init = tobool(get_env("TERRAGRUNT_DISABLE_INIT", "false"))
}

inputs = merge(
  local.stack_config.inputs,
  {
    user         = local.user
    stack        = local.stack
    default_tags = local.default_tags
    s3_prefix    = local.s3_prefix
    s3_suffix    = local.s3_suffix

    # VPC CIDRs
    app_vpc_cidr              = "100.64.0.0/16"
    debug_public_subnet_cidr  = "100.64.10.0/25"
    debug_private_subnet_cidr = "100.64.10.128/25"
  },
)

terraform {
  before_hook "Environment" {
    commands = ["init", "plan", "apply", "destroy"]
    execute  = ["echo", "STACK=${local.stack}, STAGE=${local.stack_config.inputs.stage}, AWS_REGION=${local.stack_config.inputs.region}"]
  }

  extra_arguments "init" {
    commands = ["init"]
    arguments = [
      "-upgrade=true",
      "-reconfigure",
    ]
  }

  extra_arguments "plan" {
    commands = ["plan"]
    arguments = [
      "-out=tfplan"
    ]
  }
}