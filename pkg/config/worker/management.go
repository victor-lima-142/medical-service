package worker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RabbitMQQueue struct {
	Name       string `json:"name"`
	VHost      string `json:"vhost"`
	Durable    bool   `json:"durable"`
	AutoDelete bool   `json:"auto_delete"`
	Messages   int    `json:"messages"`
}

type RabbitMQExchange struct {
	Name       string `json:"name"`
	VHost      string `json:"vhost"`
	Type       string `json:"type"`
	Durable    bool   `json:"durable"`
	AutoDelete bool   `json:"auto_delete"`
}

type SyncConfig struct {
	Queues    []string
	Exchanges []string
	Prefix    string
}

func (r *RabbitMQ) ListQueues() ([]RabbitMQQueue, error) {
	url := fmt.Sprintf("%s/api/queues", r.cfg.ManagementURL())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.SetBasicAuth(r.cfg.User, r.cfg.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar filas: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	var queues []RabbitMQQueue
	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return queues, nil
}

func (r *RabbitMQ) ListExchanges() ([]RabbitMQExchange, error) {
	url := fmt.Sprintf("%s/api/exchanges", r.cfg.ManagementURL())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.SetBasicAuth(r.cfg.User, r.cfg.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar exchanges: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	var exchanges []RabbitMQExchange
	if err := json.NewDecoder(resp.Body).Decode(&exchanges); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return exchanges, nil
}

func (r *RabbitMQ) SyncQueuesAndExchanges(config SyncConfig) error {
	desiredQueues := make(map[string]bool)
	for _, q := range config.Queues {
		desiredQueues[q] = true
		desiredQueues[q+".dlq"] = true
		for i := 1; i <= 3; i++ {
			desiredQueues[fmt.Sprintf("%s.retry.%d", q, i)] = true
		}
	}

	desiredExchanges := make(map[string]bool)
	for _, e := range config.Exchanges {
		desiredExchanges[e] = true
	}

	existingQueues, err := r.ListQueues()
	if err != nil {
		return fmt.Errorf("erro ao listar filas existentes: %w", err)
	}

	for _, q := range existingQueues {
		if strings.HasPrefix(q.Name, "amq.") {
			continue
		}

		if config.Prefix != "" && !strings.HasPrefix(q.Name, config.Prefix) {
			continue
		}

		if !desiredQueues[q.Name] {
			if _, err := r.DeleteQueue(q.Name, false, false); err != nil {
				return fmt.Errorf("erro ao deletar fila '%s': %w", q.Name, err)
			}
			fmt.Printf("  ✗ fila '%s' removida\n", q.Name)
		}
	}

	existingExchanges, err := r.ListExchanges()
	if err != nil {
		return fmt.Errorf("erro ao listar exchanges existentes: %w", err)
	}

	for _, e := range existingExchanges {
		if e.Name == "" || strings.HasPrefix(e.Name, "amq.") {
			continue
		}

		if config.Prefix != "" && !strings.HasPrefix(e.Name, config.Prefix) {
			continue
		}

		if !desiredExchanges[e.Name] {
			if err := r.DeleteExchange(e.Name, false); err != nil {
				return fmt.Errorf("erro ao deletar exchange '%s': %w", e.Name, err)
			}
			fmt.Printf("  ✗ exchange '%s' removido\n", e.Name)
		}
	}

	return nil
}

func (r *RabbitMQ) DeleteAllQueuesWithPrefix(prefix string) error {
	queues, err := r.ListQueues()
	if err != nil {
		return err
	}

	for _, q := range queues {
		if strings.HasPrefix(q.Name, prefix) {
			if _, err := r.DeleteQueue(q.Name, false, false); err != nil {
				return fmt.Errorf("erro ao deletar fila '%s': %w", q.Name, err)
			}
			fmt.Printf("  ✗ fila '%s' removida\n", q.Name)
		}
	}

	return nil
}

func (r *RabbitMQ) DeleteAllExchangesWithPrefix(prefix string) error {
	exchanges, err := r.ListExchanges()
	if err != nil {
		return err
	}

	for _, e := range exchanges {
		if e.Name != "" && !strings.HasPrefix(e.Name, "amq.") && strings.HasPrefix(e.Name, prefix) {
			if err := r.DeleteExchange(e.Name, false); err != nil {
				return fmt.Errorf("erro ao deletar exchange '%s': %w", e.Name, err)
			}
			fmt.Printf("  ✗ exchange '%s' removido\n", e.Name)
		}
	}

	return nil
}
