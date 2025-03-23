package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	DB_USER     = "admin"
	DB_PASSWORD = "admin"
	DB_NAME     = "testdb"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
)

type Request struct {
	deviceID  string
	latitude  string
	longitude string
	speed     string
	heading   string
	conn      net.Conn
}

// Variáveis de pool de conexão
var dbPool *pgxpool.Pool

// Função para criar a tabela caso ela não exista
func createTableIfNotExists() {
	conn, err := dbPool.Acquire(context.Background())
	if err != nil {
		fmt.Println("Erro ao conectar no banco:", err)
		return
	}
	defer conn.Release()

	query := `
	CREATE TABLE IF NOT EXISTS gps_data (
		id SERIAL PRIMARY KEY,
		device_id TEXT NOT NULL,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		speed DOUBLE PRECISION,
		heading DOUBLE PRECISION,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		fmt.Println("Erro ao criar tabela:", err)
		return
	}
	fmt.Println("Tabela 'gps_data' verificada/criada com sucesso!")
}

// Função para salvar os dados no banco de dados usando pool de conexões
func salvarDados(deviceID string, latitude, longitude, speed, heading string) error {
	conn, err := dbPool.Acquire(context.Background())
	if err != nil {
		return fmt.Errorf("erro ao conectar no banco: %w", err)
	}
	defer conn.Release()

	query := `INSERT INTO gps_data (device_id, latitude, longitude, speed, heading, timestamp)
              VALUES ($1, $2, $3, $4, $5, NOW())`
	_, err = conn.Exec(context.Background(), query, deviceID, latitude, longitude, speed, heading)
	if err != nil {
		return fmt.Errorf("erro ao inserir dados no banco: %w", err)
	}
	return nil
}

// Função para processar uma requisição
func processRequest(req Request) bool {
	fields := []string{req.deviceID, req.latitude, req.longitude, req.speed, req.heading}
	if len(fields) == 5 {
		// Salvar dados e retornar erro se falhar
		err := salvarDados(fields[0], fields[1], fields[2], fields[3], fields[4])
		if err != nil {
			// Enviar erro de volta para o cliente
			req.conn.Write([]byte(fmt.Sprintf("ERROR: %s\n", err.Error())))
			return false
		}
		req.conn.Write([]byte("OK\n"))
		return true
	} else {
		req.conn.Write([]byte("ERROR: Dados inválidos\n"))
		return false
	}
}

// Função para lidar com conexões
func handleConnection(conn net.Conn, requestQueue chan Request) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			// Ignorar erros de desconexão após confirmação
			if err.Error() != "EOF" && strings.Contains(err.Error(), "wsarecv") {
				return
			}
			return
		}

		fields := strings.Split(strings.TrimSpace(data), ",")
		if len(fields) == 5 {
			req := Request{
				deviceID:  fields[0],
				latitude:  fields[1],
				longitude: fields[2],
				speed:     fields[3],
				heading:   fields[4],
				conn:      conn,
			}
			requestQueue <- req
		} else {
			conn.Write([]byte("ERROR: Formato de dados inválido\n"))
		}
	}
}

// Função para processar as requisições da fila
func processQueue(requestQueue chan Request, wg *sync.WaitGroup) {
	defer wg.Done()

	for req := range requestQueue {
		if !processRequest(req) {
			fmt.Println("Erro ao processar requisição!")
		}
	}
}

func main() {
	// Iniciar o pool de conexões com o banco de dados
	connConfig, err := pgxpool.ParseConfig(fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		DB_USER, DB_PASSWORD, DB_NAME, DB_HOST, DB_PORT))
	if err != nil {
		fmt.Println("Erro ao configurar o pool de conexões:", err)
		return
	}

	// Criando o pool de conexões
	dbPool, err = pgxpool.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		fmt.Println("Erro ao criar o pool de conexões:", err)
		return
	}
	defer dbPool.Close()

	createTableIfNotExists()

	// Inicia o servidor TCP
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Servidor TCP rodando na porta 5000...")

	requestQueue := make(chan Request, 100)

	var wg sync.WaitGroup
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processQueue(requestQueue, &wg)
	}

	// Aceitar conexões
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}
		go handleConnection(conn, requestQueue)
	}
}
