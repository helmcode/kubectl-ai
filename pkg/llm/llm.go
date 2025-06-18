package llm

type LLM interface {
    Chat(prompt string) (string, error)
}
