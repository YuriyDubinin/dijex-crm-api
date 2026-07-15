package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// ───────────────────────── Opaque-токен ─────────────────────────
// «Opaque» значит «непрозрачный»: токен — просто случайные байты в base64url,
// внутри него никаких данных. Все атрибуты лежат в auth_tokens (customer_id,
// user_id, expires_at и т.д.). Это основной токен сессии.

// DefaultTokenByteLength — длина опакового токена в байтах.
// 32 байта = 256 бит энтропии — с большим запасом против перебора.
const DefaultTokenByteLength = 32

// GenerateOpaqueToken возвращает криптостойкую случайную строку длиной
// `byteLen` байт в URL-safe base64 без паддинга. При byteLen <= 0 берёт
// DefaultTokenByteLength.
//
// Применение:
//   - сгенерировать токен при логине;
//   - вернуть клиенту в открытом виде (один раз, в ответе на login);
//   - в auth_tokens.token_hash положить HashToken(<этот токен>);
//   - при последующих запросах клиент шлёт сырой токен в заголовке
//     Authorization: Bearer ... — сервис хэширует и ищет в БД по token_hash.
func GenerateOpaqueToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = DefaultTokenByteLength
	}
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto: generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken — детерминированный SHA-256 хэш (hex) сырого токена.
// Используется и при записи в auth_tokens.token_hash, и при поиске по нему.
// Помощник один на оба слоя — критически важно, чтобы алгоритм совпадал.
//
// Внимание: для ПАРОЛЕЙ этот хэш НЕ годится (SHA-256 слишком быстр для
// перебора). Пароли — через PasswordHasher (bcrypt).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
