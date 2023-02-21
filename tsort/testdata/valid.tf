variable "kubernetes_pipeline_roles" {
  description = "IAM roles for pipelines required access to EKS."
  type = list(object({
    rolearn    = string
    namespaces = list(string)
  }))
  default = []
}




variable "kubernetes_pipeline_users" {
  description = "IAM users for pipelines required access to EKS."
  type = list(object({
    userarn    = string
    namespaces = list(string)
  }))
  default = []
}


variable "external_dns_additional_managed_zones" {
  description = "Additional managed zones for external-dns."
  type        = list(string)
  default     = []
}

variable "aws_profile" {
  description = "The aws profile name, used when creating the kubeconfig file."
}

variable "additional_userdata" {
  default = ""
}






variable "eks_shared_namespaces" {
  description = "Namespaces to be shared between teams."
  type        = map(list(string))
  default = {
    dns        = ["external-dns"]
    infra      = ["infra-shared"]
    logging    = ["logging"]
    monitoring = ["infra-monitoring"]
    ingress    = ["infra-ingress"]
    argo       = ["argo"]
    newrelic   = ["newrelic"]
  }
}
