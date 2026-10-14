package stores

import (
	"database/sql"
	"encoding/json"

	"coachia-backend/ai"
)

type ChatStore struct {
	db *sql.DB
}

func NewChatStore(db *sql.DB) *ChatStore {
	return &ChatStore{db: db}
}

func (s *ChatStore) Append(userID int64, role string, content []ai.ContentBlock) error {
	data, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO chat_messages (user_id, role, content) VALUES (?, ?, ?)`, userID, role, string(data))
	return err
}

func (s *ChatStore) List(userID int64) ([]ai.Message, error) {
	rows, err := s.db.Query(`SELECT role, content FROM chat_messages WHERE user_id = ? ORDER BY id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ai.Message
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, err
		}
		var blocks []ai.ContentBlock
		if err := json.Unmarshal([]byte(content), &blocks); err != nil {
			return nil, err
		}
		messages = append(messages, ai.Message{Role: role, Content: blocks})
	}
	return messages, rows.Err()
}
