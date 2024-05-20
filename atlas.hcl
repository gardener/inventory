data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./internal/cmd/atlas-loader",
  ]
}

variable "url" {
  type    = string
  default = "postgres://inventory:inventoryp4ssw0rd@localhost:5432/inventory?sslmode=disable"
}

env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/16.2-bookworm/inventory"
  url = var.url
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
