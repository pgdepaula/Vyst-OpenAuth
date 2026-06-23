provider "aws" {
  region = "us-east-1"
}

resource "aws_ecr_repository" "vyst_identity" {
  name = "vyst-identity"
}
