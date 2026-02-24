package redisstate

import (
	"LoudQuestionBot/internal/domain/repository"
	"LoudQuestionBot/internal/domain/schema"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 24 * time.Hour

type FormStateRepo struct {
	client *redis.Client
}

var _ repository.FormStateRepository = (*FormStateRepo)(nil)

func NewFormStateRepo(client *redis.Client) *FormStateRepo {
	return &FormStateRepo{client: client}
}

func (r *FormStateRepo) Get(ctx context.Context, userID int64) (schema.FormState, bool, error) {
	v, err := r.client.Get(ctx, formKey(userID)).Result()
	if err == redis.Nil {
		return schema.FormState{}, false, nil
	}
	if err != nil {
		return schema.FormState{}, false, err
	}

	var state schema.FormState
	if err := json.Unmarshal([]byte(v), &state); err != nil {
		return schema.FormState{}, false, err
	}
	return state, true, nil
}

func (r *FormStateRepo) Set(ctx context.Context, userID int64, state schema.FormState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, formKey(userID), b, ttl).Err()
}

func (r *FormStateRepo) Delete(ctx context.Context, userID int64) error {
	return r.client.Del(ctx, formKey(userID)).Err()
}

func formKey(userID int64) string {
	return fmt.Sprintf("form:%d", userID)
}
