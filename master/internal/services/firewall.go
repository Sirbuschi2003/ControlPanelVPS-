package services

import (
	"context"
	"fmt"

	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/agent"
	"github.com/Sirbuschi2003/ControlPanelVPS/master/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FirewallService manages firewall rules via the remote agent.
type FirewallService struct {
	db *pgxpool.Pool
}

// NewFirewallService creates a new FirewallService.
func NewFirewallService(db *pgxpool.Pool) *FirewallService {
	return &FirewallService{db: db}
}

func (s *FirewallService) agentFor(ctx context.Context, serverID string) (*agent.AgentClient, error) {
	var agentURL, agentToken string
	err := s.db.QueryRow(ctx, `SELECT agent_url, agent_token FROM servers WHERE id = $1`, serverID).
		Scan(&agentURL, &agentToken)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}
	return agent.NewAgentClient(agentURL, agentToken), nil
}

// List returns all firewall rules for a server.
func (s *FirewallService) List(ctx context.Context, serverID string) ([]models.FirewallRule, error) {
	query := `SELECT id, server_id, rule_order, action, direction, protocol, source,
		       dest_port, comment, enabled, created_at
		FROM firewall_rules ORDER BY server_id, rule_order ASC`
	args := []interface{}{}
	if serverID != "" {
		query = `SELECT id, server_id, rule_order, action, direction, protocol, source,
		       dest_port, comment, enabled, created_at
		FROM firewall_rules WHERE server_id = $1 ORDER BY rule_order ASC`
		args = append(args, serverID)
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query firewall rules: %w", err)
	}
	defer rows.Close()

	var rules []models.FirewallRule
	for rows.Next() {
		var r models.FirewallRule
		if err := rows.Scan(
			&r.ID, &r.ServerID, &r.RuleOrder, &r.Action, &r.Direction, &r.Protocol, &r.Source,
			&r.DestPort, &r.Comment, &r.Enabled, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan firewall rule: %w", err)
		}
		rules = append(rules, r)
	}
	if rules == nil {
		rules = []models.FirewallRule{}
	}
	return rules, nil
}

type agentFirewallRulePayload struct {
	Action    string  `json:"action"`
	Direction string  `json:"direction"`
	Protocol  string  `json:"protocol"`
	Source    string  `json:"source"`
	DestPort  *string `json:"dest_port,omitempty"`
	Comment   *string `json:"comment,omitempty"`
	Order     int     `json:"order"`
}

// Create adds a new firewall rule on the agent and stores it in the database.
func (s *FirewallService) Create(ctx context.Context, serverID, action, direction, protocol, source string, destPort, comment *string, order int) (*models.FirewallRule, error) {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return nil, err
	}

	payload := agentFirewallRulePayload{
		Action:    action,
		Direction: direction,
		Protocol:  protocol,
		Source:    source,
		DestPort:  destPort,
		Comment:   comment,
		Order:     order,
	}

	_, err = ac.Post(ctx, "/firewall/rules", payload)
	if err != nil {
		return nil, fmt.Errorf("agent create firewall rule: %w", err)
	}

	var r models.FirewallRule
	err = s.db.QueryRow(ctx, `
		INSERT INTO firewall_rules (server_id, rule_order, action, direction, protocol, source, dest_port, comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, server_id, rule_order, action, direction, protocol, source,
		          dest_port, comment, enabled, created_at
	`, serverID, order, action, direction, protocol, source, destPort, comment).Scan(
		&r.ID, &r.ServerID, &r.RuleOrder, &r.Action, &r.Direction, &r.Protocol, &r.Source,
		&r.DestPort, &r.Comment, &r.Enabled, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert firewall rule: %w", err)
	}

	// Reload firewall after adding rule
	_ = s.reloadAgent(ctx, ac)

	return &r, nil
}

// Delete removes a firewall rule from the agent and the database.
func (s *FirewallService) Delete(ctx context.Context, id string) error {
	var r models.FirewallRule
	err := s.db.QueryRow(ctx, `
		SELECT id, server_id, rule_order, action, direction, protocol, source,
		       dest_port, comment, enabled, created_at
		FROM firewall_rules WHERE id = $1
	`, id).Scan(
		&r.ID, &r.ServerID, &r.RuleOrder, &r.Action, &r.Direction, &r.Protocol, &r.Source,
		&r.DestPort, &r.Comment, &r.Enabled, &r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("firewall rule not found: %w", err)
	}

	ac, err := s.agentFor(ctx, r.ServerID)
	if err != nil {
		return err
	}

	// Send rule details to identify which rule to delete
	delPayload := agentFirewallRulePayload{
		Action:    r.Action,
		Direction: r.Direction,
		Protocol:  r.Protocol,
		Source:    r.Source,
		DestPort:  r.DestPort,
		Comment:   r.Comment,
		Order:     r.RuleOrder,
	}

	_, err = ac.Post(ctx, "/firewall/rules/delete", delPayload)
	if err != nil {
		return fmt.Errorf("agent delete firewall rule: %w", err)
	}

	_, err = s.db.Exec(ctx, `DELETE FROM firewall_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete firewall rule from db: %w", err)
	}

	_ = s.reloadAgent(ctx, ac)
	return nil
}

// Toggle enables or disables a firewall rule.
func (s *FirewallService) Toggle(ctx context.Context, id string, enabled bool) error {
	_, err := s.db.Exec(ctx, `UPDATE firewall_rules SET enabled = $1 WHERE id = $2`, enabled, id)
	if err != nil {
		return fmt.Errorf("toggle firewall rule: %w", err)
	}
	return nil
}

// Reload applies the current firewall ruleset on the agent.
func (s *FirewallService) Reload(ctx context.Context, serverID string) error {
	ac, err := s.agentFor(ctx, serverID)
	if err != nil {
		return err
	}
	return s.reloadAgent(ctx, ac)
}

func (s *FirewallService) reloadAgent(ctx context.Context, ac *agent.AgentClient) error {
	_, err := ac.Post(ctx, "/firewall/reload", nil)
	if err != nil {
		return fmt.Errorf("agent reload firewall: %w", err)
	}
	return nil
}
