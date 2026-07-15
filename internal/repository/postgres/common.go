package postgres

import "encoding/json"

// uniqueViolationCode — SQLSTATE нарушения уникального ограничения (23505).
// Репозитории маппят его в domain.ErrAlreadyExists.
const uniqueViolationCode = "23505"

// jsonbArg приводит JSON-атрибуты к аргументу для jsonb-колонки. Пустое
// значение превращается в пустой объект '{}'. Использовать вместе с
// приведением $n::jsonb в SQL (иначе pgx закодирует []byte как bytea).
func jsonbArg(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

// jsonbArgOrNil — как jsonbArg, но пустое значение даёт NULL (для nullable
// jsonb-колонок вроде clients.address).
func jsonbArgOrNil(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

// sortDir возвращает "ASC"/"DESC" по флагу.
func sortDir(desc bool) string {
	if desc {
		return "DESC"
	}
	return "ASC"
}
