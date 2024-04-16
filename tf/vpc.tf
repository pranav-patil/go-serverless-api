# Create Application VPC, Subnets and Security Group

# Get the available Availability Zones from AWS
data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  local_azs = data.aws_availability_zones.available.names
  main_azs  = slice(data.aws_availability_zones.available.names, 0, 3)

  public_subnets = [
    cidrsubnet(var.app_vpc_cidr, 8, 1), # 10.0.1.0/24
    cidrsubnet(var.app_vpc_cidr, 8, 2)  # 10.0.2.0/24
  ]

  private_subnets = [
    cidrsubnet(var.app_vpc_cidr, 2, 2), # 10.0.4.0/24
    cidrsubnet(var.app_vpc_cidr, 2, 3)  # "10.0.5.0/24
  ]

  intra_nat_subnets = [
    cidrsubnet(var.intra_nat_cidr, 1, 0),
    cidrsubnet(var.intra_nat_cidr, 1, 1)
  ]

}

# Create VPC
module "app_vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name                  = "app-vpc"
  cidr                  = var.intra_nat_cidr
  secondary_cidr_blocks = [var.app_vpc_cidr]

  azs             = slice(data.aws_availability_zones.available.names, 0, 2)
  private_subnets = local.private_subnets
  public_subnets  = local.public_subnets
  intra_subnets   = local.intra_nat_subnets

  # create public NAT per AZs
  enable_nat_gateway     = true
  single_nat_gateway     = false
  one_nat_gateway_per_az = true

  enable_dns_hostnames = true
  default_security_group_egress = [
    {
      description = "Allow all outbound traffic"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      cidr_blocks = "0.0.0.0/0"
    }
  ]
}

# private NAT
resource "aws_nat_gateway" "navigate_transit_gateway" {
  count = length(module.app_vpc.intra_subnets)

  connectivity_type = "private"
  subnet_id         = module.app_vpc.intra_subnets[count.index]

  tags = local.tags

  lifecycle {
    ignore_changes = [tags]
  }
}

# private subnet to intra NAT gateway
resource "aws_route" "private_to_intra_nat" {
  for_each = { for key, value in module.app_vpc.private_route_table_ids : key => value }

  route_table_id         = each.value
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.navigate_transit_gateway[each.key].id
}

// VPC endpoint rule
resource "aws_security_group_rule" "vpc_to_endpoint" {
  description       = "VPC to Endpoint"
  type              = "ingress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = [var.app_vpc_cidr]
  security_group_id = module.app_vpc.default_security_group_id
}

resource "aws_vpc_endpoint" "ssm" {
  vpc_id       = module.app_vpc.vpc_id
  service_name = "com.amazonaws.${var.region}.ssm"
  subnet_ids   = module.app_vpc.public_subnets

  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  security_group_ids = [
    module.app_vpc.default_security_group_id
  ]

  tags = {
    Name = "${module.app_vpc.name}-ssm"
  }
}

resource "aws_vpc_endpoint" "sts" {
  vpc_id       = module.app_vpc.vpc_id
  service_name = "com.amazonaws.${var.region}.sts"
  subnet_ids   = module.app_vpc.public_subnets

  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  security_group_ids = [
    module.app_vpc.default_security_group_id
  ]

  tags = {
    Name = "${module.app_vpc.name}-sts"
  }
}

resource "aws_vpc_endpoint" "secretsmanager" {
  vpc_id       = module.app_vpc.vpc_id
  service_name = "com.amazonaws.${var.region}.secretsmanager"
  subnet_ids   = module.app_vpc.public_subnets

  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  security_group_ids = [
    module.app_vpc.default_security_group_id
  ]

  tags = {
    Name = "${module.app_vpc.name}-secretsmanager"
  }
}

resource "aws_vpc_endpoint" "kms" {
  vpc_id       = module.app_vpc.vpc_id
  service_name = "com.amazonaws.${var.region}.kms"
  subnet_ids   = module.app_vpc.public_subnets

  vpc_endpoint_type   = "Interface"
  private_dns_enabled = true

  security_group_ids = [
    module.app_vpc.default_security_group_id
  ]

  tags = {
    Name = "${module.app_vpc.name}-kms"
  }
}



