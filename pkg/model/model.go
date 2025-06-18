package model

type Analysis struct {
    Problem      string     `json:"problem"`
    RootCause    string     `json:"root_cause"`
    Severity     string     `json:"severity"`
    Issues       []Issue    `json:"issues"`
    Suggestions  []Suggestion `json:"suggestions"`
    QuickFix     string     `json:"quick_fix,omitempty"`
    FullAnalysis string     `json:"full_analysis"`
}

type Issue struct {
    Component   string `json:"component"`
    Severity    string `json:"severity"`
    Description string `json:"description"`
    Evidence    string `json:"evidence,omitempty"`
}

type Suggestion struct {
    Priority    string `json:"priority"`
    Action      string `json:"action"`
    Command     string `json:"command,omitempty"`
    Explanation string `json:"explanation"`
}
