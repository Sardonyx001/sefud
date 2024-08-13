package config

type DBConfig struct {
	User     string
	Password string
	Name     string
	Host     string
	Port     string
}

func LoadDBConfig() DBConfig {
	return DBConfig{
		User:     GetEnv("SEFUD_DB_USER", "user"),
		Password: GetEnv("SEFUD_DB_PASSWORD", "password"),
		Name:     GetEnv("SEFUD_DB_NAME", "sefud"),
		Host:     GetEnv("SEFUD_DB_HOST", "postgres"),
		Port:     GetEnv("SEFUD_DB_PORT", "5432"),
	}
}
