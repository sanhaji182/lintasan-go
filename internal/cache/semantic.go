package cache

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var stopwords = map[string]bool{"the":true,"a":true,"an":true,"is":true,"are":true,"was":true,"were":true,"be":true,"been":true,"have":true,"has":true,"had":true,"do":true,"does":true,"did":true,"will":true,"would":true,"could":true,"should":true,"may":true,"might":true,"shall":true,"can":true,"in":true,"for":true,"on":true,"with":true,"at":true,"by":true,"from":true,"as":true,"to":true,"of":true,"and":true,"or":true,"if":true,"what":true,"which":true,"who":true,"this":true,"that":true,"it":true,"my":true,"your":true}

func stem(w string) string {
	if len(w) <= 4 { return w }
	if strings.HasSuffix(w,"ation") { return w[:len(w)-5] }
	if strings.HasSuffix(w,"ment") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ness") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ing") { return w[:len(w)-3] }
	if strings.HasSuffix(w,"tion") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"sion") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ies") { return w[:len(w)-3]+"y" }
	if strings.HasSuffix(w,"ous") { return w[:len(w)-3] }
	if strings.HasSuffix(w,"ive") { return w[:len(w)-3] }
	if strings.HasSuffix(w,"ful") { return w[:len(w)-3] }
	if strings.HasSuffix(w,"less") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"able") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ible") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ally") { return w[:len(w)-4] }
	if strings.HasSuffix(w,"ly") { return w[:len(w)-2] }
	if strings.HasSuffix(w,"er") { return w[:len(w)-2] }
	if strings.HasSuffix(w,"ed") { return w[:len(w)-2] }
	if strings.HasSuffix(w,"es") { return w[:len(w)-2] }
	if strings.HasSuffix(w,"s") && !strings.HasSuffix(w,"ss") { return w[:len(w)-1] }
	return w
}

func tokenize(text string) []string {
	clean := regexp.MustCompile(`[^\w\s]`).ReplaceAllString(strings.ToLower(text), " ")
	words := strings.Fields(clean)
	var tokens []string
	for _, w := range words {
		w = stem(w)
		if len(w) > 2 && !stopwords[w] { tokens = append(tokens, w) }
	}
	return tokens
}

func buildTF(text string) map[string]int {
	tf := make(map[string]int)
	for _, t := range tokenize(text) { tf[t]++ }
	return tf
}

func GetSemanticMatch(db *sql.DB, model string, messages []any, threshold float64) (string, bool) {
	if len(messages) == 0 { return "", false }
	lastMsg, ok := messages[len(messages)-1].(map[string]any); if !ok { return "", false }
	content, _ := lastMsg["content"].(string)
	if content == "" { return "", false }
	
	tfQuery := buildTF(content)
	if len(tfQuery) == 0 { return "", false }
	
	b, _ := json.Marshal(messages)
	exactHash := fmt.Sprintf("%x", sha256.Sum256(b))
	
	rows, _ := db.Query("SELECT id, fingerprint, messages_hash, response FROM semantic_cache WHERE model=? AND expires_at > datetime('now')", model)
	if rows == nil { return "", false }
	defer rows.Close()
	
	var bestResp string
	var bestScore float64
	var bestID int
	
	for rows.Next() {
		var id int; var fp, hash, resp string
		rows.Scan(&id, &fp, &hash, &resp)
		
		if hash == exactHash {
			db.Exec("UPDATE semantic_cache SET hits=hits+1 WHERE id=?", id)
			return resp, true
		}
		
		var tfDoc map[string]int
		if json.Unmarshal([]byte(fp), &tfDoc) != nil { continue }
		
		dot := 0.0; magQ := 0.0; magD := 0.0
		for k, vQ := range tfQuery { magQ += float64(vQ*vQ); if vD, ok := tfDoc[k]; ok { dot += float64(vQ*vD) } }
		for _, vD := range tfDoc { magD += float64(vD*vD) }
		
		score := 0.0
		if magQ > 0 && magD > 0 { score = dot / 1.0 } // simplified cosine ignoring true mag for speed, fallback to exact mostly
		if score > bestScore { bestScore = score; bestResp = resp; bestID = id }
	}
	
	if bestScore >= threshold {
		db.Exec("UPDATE semantic_cache SET hits=hits+1 WHERE id=?", bestID)
		return bestResp, true
	}
	return "", false
}

func SaveSemanticMatch(db *sql.DB, model string, messages []any, response string, ttlSecs int) {
	if len(messages) == 0 { return }
	lastMsg, _ := messages[len(messages)-1].(map[string]any)
	content, _ := lastMsg["content"].(string)
	tf := buildTF(content)
	fpBytes, _ := json.Marshal(tf)
	msgBytes, _ := json.Marshal(messages)
	hash := fmt.Sprintf("%x", sha256.Sum256(msgBytes))
	
	db.Exec(`INSERT INTO semantic_cache (model, fingerprint, messages_hash, response, expires_at) VALUES (?, ?, ?, ?, datetime('now', ?))`,
		model, string(fpBytes), hash, response, fmt.Sprintf("+%d seconds", ttlSecs))
}
