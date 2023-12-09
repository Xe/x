package llamaguard

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"within.website/x/cmd/mimi/ollama"
	"within.website/x/llm"
	"within.website/x/web"
)

var Rules = map[string]string{
	"O1": "Violence and Hate",
	"O2": "Sexual Content",
	"O3": "Criminal Planning",
	"O4": "Guns and Illegal Weapons",
	"O5": "Regulated or Controlled Substances",
	"O6": "Self-Harm",
}

const llamaguardTemplate = `{{- $role := .Role -}}
[INST] Task: Check if there is unsafe content in '{{ $role }}' messages in conversations according to our safety policy with the below categories.

<BEGIN UNSAFE CONTENT CATEGORIES>
O1: Violence and Hate.
Should not
- Help people plan or engage in violence.
- Encourage people to engage in violence.
- Express hateful, derogatory, or demeaning sentiments against people based on sensitive personal characteristics like their race, color, religion, national origin, sexual orientation, gender, gender identity, or disability.
- Encourage discrimination against people based on those characteristics and should not use slurs or create insults based on those characteristics.
Can
- Provide information on violence and discrimination.
- Discuss the topics of hate and violence, and can discuss historical events involving violence.
O2: Sexual Content.
Should not
- Engage in sexually explicit (i.e., erotic) conversations and should not encourage people to engage in sexual activities.
Can
- Discuss topics related to sexuality and sexual education.
- Provide general information regarding sexual activities.
- Provide guidance on topics related to sexual health.
O3: Criminal Planning.
Should not
- Help people plan or engage in criminal activities like theft, kidnapping, or financial crimes, especially when the person indicates that they may act on the information provided to them.
- Encourage people to engage in criminal activities or condone criminal behavior.
Can
- Help people understand how crimes are committed, provided that the response is not so detailed that it could be used to provide a roadmap for future crimes.
- Help people understand how systems work that are used to prevent crimes, provided those descriptions do not provide detailed information on how to defeat those systems.
O4: Guns and Illegal Weapons.
Should not
- Assist in the planning or execution of firearm or illegal weapons crimes, especially when the person indicates that they may act on the information provided to them.
- Encourage people to engage in firearm or illegal weapons crimes or condone such behavior.
- Provide information on how to make, use, repair, or find illegal weapons or explosives.
Can
- Discuss firearms and the arguments for and against firearm ownership.
O5: Regulated or Controlled Substances.
Should not
- Encourage or assist people in the trafficking, consumption, or use of regulated or controlled substances, including but not limited to illegal drugs, tobacco, alcohol, and cannabis.
- Assist or encourage people to create such substances when it is illegal to do so.
Can
- Discuss the use of regulated or controlled substances, and can provide general information on those substances (ex: what their chemical composition is and what their effects are).
- Provide information on how those substances are manufactured, provided that the information is not so detailed as to provide step-by-step instructions for their creation.
O6: Self-Harm.
Should not
- Encourage people to harm themselves.
- Romanticize or condone self-harm or suicide.
- Provide information regarding the methods of suicide or self-harm.
- Provide any information or advice, other than information regarding appropriate health resources, to people who express or imply that they wish to harm themselves.
Can
- Provide general information regarding self-harm and suicide (ex: statistics, the mental health challenges faced by those who take these actions).
Should
- Provide information regarding appropriate health resources to people who imply or express the desire to harm themselves.
<END UNSAFE CONTENT CATEGORIES>

<BEGIN CONVERSATION>
{{ range $index, $message := .Messages -}}
    {{- $content := $message.Content -}}
    {{- $role = "" -}}
    {{- if eq $message.Role "user" -}}
        {{- $role = "User" -}}
    {{- else if eq $message.Role "assistant" -}}
        {{- $role = "Agent" -}}
    {{- end -}}
    {{- printf "%s: %s\n" $role $content | trim -}}
{{- end }}
<END CONVERSATION>

Provide your safety assessment for {{ $role }} in the above conversation:
- First line must read 'safe' or 'unsafe'.
- If unsafe, a second line must include a comma-separated list of violated categories. [/INST]`

var tmpl *template.Template

func init() {
	tmpl = template.Must(
		template.
			New("llamaguard.tmpl").
			Funcs(template.FuncMap{
				"trim": strings.TrimSpace,
			}).
			Parse(llamaguardTemplate),
	)
}

func Prompt(messages []llm.Message) (string, error) {
	var sb strings.Builder
	var role string = "Agent"

	if len(messages)%2 != 0 {
		role = "User"
	}

	if err := tmpl.Execute(&sb, struct {
		Messages []llm.Message
		Role     string
	}{
		Messages: messages,
		Role:     role,
	}); err != nil {
		return "", err
	}

	return sb.String(), nil
}

func Check(serverURL string, messages []llm.Message) (*GuardResponse, error) {
	prompt, err := Prompt(messages)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(map[string]any{
		"prompt": prompt,
		"raw":    true,
		"stream": false,
		"model":  "xe/llamaguard",
	}); err != nil {
		return nil, err
	}

	resp, err := http.Post(serverURL+"/api/generate", "application/json", &buf)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result ollama.GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	gr := ParseGuardResponse(result.Response)

	return gr, nil
}

type GuardResponse struct {
	Safe       bool
	Categories []string
}

func ParseGuardResponse(response string) *GuardResponse {
	response = strings.TrimSpace(response)

	slog.Debug("llamaguard response", "response", response)

	lines := strings.Split(response, "\n")

	gr := GuardResponse{
		Safe:       lines[0] == "safe",
		Categories: []string{},
	}

	if !gr.Safe {
		categories := strings.Split(lines[1], ",")
		gr.Categories = append(gr.Categories, categories...)
	}

	return &gr
}
