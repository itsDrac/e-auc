package main

import (
	"errors"
	"fmt"
	"net/http"
)

type Application struct {
	serv http.Server
}

type Config struct {
	addr string
	DB DBConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func (d *DBConfig) ConnectionString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", d.User, d.Password, d.Host, d.Port, d.Name)
}

func (app *Application) mount() http.Handler {
	return nil
}

func (app *Application) run(_ http.Handler) error {
	return errors.New("Not implemented")
}