data "external_schema" "app" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./models",
    "--dialect", "sqlite",
  ]
}

env "app" {
  src = data.external_schema.app.url
  dev = "sqlite://dbs/app_dev.db?mode=memory&cache=shared&_fk=1"
  url = "sqlite://dbs/app.db?mode=memory&cache=shared&_fk=1"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
