variable "nodes" {
  type = map(object({
    location = string
  }))
}

variable "flake" {

}
