package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Configuração do endereço do servidor
const serverAddr = "127.0.0.1:5000" // Endereço do servidor
const totalRequests = 5000          // Número total de requisições

// Função para gerar uma latitude aleatória dentro de uma faixa de coordenadas
func generateLatitude() string {
	// Gera um número aleatório entre -90 e 90 (faixa de latitudes)
	return fmt.Sprintf("%.4f", rand.Float64()*180-90)
}

// Função para gerar uma longitude aleatória dentro de uma faixa de coordenadas
func generateLongitude() string {
	// Gera um número aleatório entre -180 e 180 (faixa de longitudes)
	return fmt.Sprintf("%.4f", rand.Float64()*360-180)
}

// Função para gerar uma velocidade aleatória entre 0 e 120 km/h
func generateSpeed() string {
	return fmt.Sprintf("%.1f", rand.Float64()*120)
}

// Função para gerar uma direção aleatória (heading) entre 0 e 360 graus
func generateHeading() string {
	return fmt.Sprintf("%.1f", rand.Float64()*360)
}

// Função para enviar dados para o servidor
func sendRequest(deviceID, latitude, longitude, speed, heading string, conn net.Conn, wg *sync.WaitGroup, successCount *int64, errorCount *int64) {
	defer wg.Done() // Decrementa o contador do WaitGroup quando terminar

	// Formato de mensagem para enviar
	message := fmt.Sprintf("%s, %s, %s, %s, %s\n", deviceID, latitude, longitude, speed, heading)

	// Enviar os dados para o servidor
	_, err := fmt.Fprintf(conn, "%s", message)
	if err != nil {
		atomic.AddInt64(errorCount, 1) // Incrementa de forma atômica
		fmt.Println("Erro ao enviar dados:", err)
		return
	}

	// Ler a resposta do servidor
	response, _ := bufio.NewReader(conn).ReadString('\n')
	if response != "OK\n" {
		atomic.AddInt64(errorCount, 1) // Incrementa de forma atômica
		fmt.Println("Resposta inesperada do servidor:", response)
		return
	}

	// Se a requisição foi bem-sucedida
	atomic.AddInt64(successCount, 1) // Incrementa de forma atômica
}

// Função para criar e conectar ao servidor
func createConnection() (net.Conn, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar: %v", err)
	}
	return conn, nil
}

// Função para enviar múltiplas requisições simultaneamente
func sendMultipleRequests() (int64, int64) {
	var wg sync.WaitGroup
	var successCount, errorCount int64

	// Limitar o número de goroutines
	wg.Add(totalRequests)

	// Enviar as requisições em paralelo
	for i := 0; i < totalRequests; i++ {
		// Criar uma nova conexão para cada requisição (isso simula 500 dispositivos diferentes)
		conn, err := createConnection()
		if err != nil {
			fmt.Println("Erro ao criar conexão:", err)
			wg.Done()
			continue
		}

		// Gerar dados aleatórios para a requisição
		deviceID := fmt.Sprintf("DEVICE%d", i+1)
		latitude := generateLatitude()
		longitude := generateLongitude()
		speed := generateSpeed()
		heading := generateHeading()

		// Enviar a requisição em uma goroutine
		go sendRequest(deviceID, latitude, longitude, speed, heading, conn, &wg, &successCount, &errorCount)
	}

	// Esperar até que todas as goroutines tenham terminado
	wg.Wait()

	// Retorna o número de sucessos e erros
	return successCount, errorCount
}

// Função para enviar as requisições
func startSendingRequests() {
	// Enviar 500 requisições simultâneas
	fmt.Printf("Enviando %d requisições simultâneas...", totalRequests)
	startTime := time.Now()
	success, errors := sendMultipleRequests()
	duration := time.Since(startTime)
	fmt.Printf("Envio de %d requisições concluído! Tempo de execução: %v\n", totalRequests, duration)
	fmt.Printf("Dados enviados com sucesso: %d\n", success)
	fmt.Printf("Erros ao enviar dados: %d\n", errors)
}

func main() {
	// Configuração do rand.Seed para gerar números aleatórios diferentes a cada execução
	rand.Seed(time.Now().UnixNano())

	// Iniciar o envio das 500 requisições simultâneas
	startSendingRequests()
}
