package sqlstore

import (
	"context"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
)

func init() {
	bus.AddHandler("sql", GetApiKeys)
	bus.AddHandler("sql", GetApiKeyById)
	bus.AddHandler("sql", GetApiKeyByName)
	bus.AddHandlerCtx("sql", DeleteApiKeyCtx)
	bus.AddHandler("sql", AddApiKey)
}

func GetApiKeys(query *m.GetApiKeysQuery) error {
	now := "'" + time.Now().Format("2006-01-02T15:04:05") + "'"
	hasNotExpired := dialect.ToTimestamp("expires") + " >= " + dialect.ToTimestamp(now)
	sess := x.Limit(100, 0).Where("org_id=? and ( expires IS NULL or "+hasNotExpired+")",
		query.OrgId).Asc("name")
	if query.IncludeInvalid {
		sess = x.Limit(100, 0).Where("org_id=?", query.OrgId).Asc("name")
	}

	query.Result = make([]*m.ApiKey, 0)
	return sess.Find(&query.Result)
}

func DeleteApiKeyCtx(ctx context.Context, cmd *m.DeleteApiKeyCommand) error {
	return withDbSession(ctx, func(sess *DBSession) error {
		var rawSql = "DELETE FROM api_key WHERE id=? and org_id=?"
		_, err := sess.Exec(rawSql, cmd.Id, cmd.OrgId)
		return err
	})
}

func AddApiKey(cmd *m.AddApiKeyCommand) error {
	return inTransaction(func(sess *DBSession) error {
		updated := time.Now()
		var expires time.Time
		if cmd.SecondsToLive > 0 {
			expires = updated.Add(time.Second * time.Duration(cmd.SecondsToLive))
		}
		t := m.ApiKey{
			OrgId:   cmd.OrgId,
			Name:    cmd.Name,
			Role:    cmd.Role,
			Key:     cmd.Key,
			Created: updated,
			Updated: updated,
			Expires: expires,
		}

		if _, err := sess.Insert(&t); err != nil {
			return err
		}
		cmd.Result = &t
		return nil
	})
}

func GetApiKeyById(query *m.GetApiKeyByIdQuery) error {
	var apikey m.ApiKey
	has, err := x.Id(query.ApiKeyId).Get(&apikey)

	if err != nil {
		return err
	} else if !has {
		return m.ErrInvalidApiKey
	}

	query.Result = &apikey
	return nil
}

func GetApiKeyByName(query *m.GetApiKeyByNameQuery) error {
	var apikey m.ApiKey
	has, err := x.Where("org_id=? AND name=?", query.OrgId, query.KeyName).Get(&apikey)

	if err != nil {
		return err
	} else if !has {
		return m.ErrInvalidApiKey
	}

	query.Result = &apikey
	return nil
}
