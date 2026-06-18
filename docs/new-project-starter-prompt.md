# Starter Prompt — Novo Projeto [PROJECT_NAME]

> Cole este prompt integralmente no início de uma nova conversa no Claude Code (Opus 4.8 ou superior) ou no Antigravity CLI (agy).
> Substitua [PROJECT_NAME] pelo nome definitivo do projeto antes de usar.

---

## Contexto pessoal

Meu nome é Victor Gonçalves. Sou desenvolvedor de software e uso Claude Code e Antigravity CLI (agy) como agentes de desenvolvimento. Tenho outros projetos ativos (hf-transaction-service, hf-income-service) onde estabeleci um fluxo completo de desenvolvimento com FDDs, Jira, Confluence e skills personalizadas — esse mesmo fluxo deve ser adotado aqui desde o início.

---

## O projeto

Quero construir um **AI Meeting Copilot + Interview Assistant** para uso pessoal (não comercial). O produto é um aplicativo desktop que:

1. **Grava reuniões de forma invisível** — a janela não aparece ao compartilhar a tela (`setContentProtection` no Electron), funciona em Zoom, Google Meet, Teams e qualquer ferramenta
2. **Transcreve áudio em tempo real** usando Whisper local (acelerado por GPU — RTX 2060, CUDA)
3. **Documenta reuniões automaticamente** — gera ata estruturada, pontos de ação e resumo executivo em Markdown
4. **Auxilia em desafios técnicos durante entrevistas** — captura a tela com o problema de código, envia para uma LLM junto com o contexto da transcrição e exibe a solução na janela invisível
5. **É agnóstico de LLM** — suporta Ollama local (Llama 3, Phi-3, Qwen) e provedores de nuvem (OpenAI, Anthropic Claude, Groq, Gemini) com chaves configuráveis pelo usuário

**Plataformas alvo:** macOS, Linux, Windows (com comportamento idêntico).

**Problema crítico no macOS:** capturar o áudio do sistema (não só o microfone) exige permissões especiais via ScreenCaptureKit — já resolvido pelo Screenpipe. Não vamos reescrever isso; vamos integrar com o Screenpipe via REST API local.

---

## Hardware do usuário

- CPU: Intel Core i9
- RAM: 32 GB
- GPU: NVIDIA RTX 2060 (6 GB VRAM, suporte a CUDA)
- OS primário de desenvolvimento: Linux (com uso em Mac e Windows)

Modelos Whisper recomendados para esse hardware: `small` (2 GB VRAM) ou `medium` (5 GB VRAM).
Modelos LLM locais: Llama 3.2 3B ou Phi-3.5-mini cabem confortavelmente junto com o Whisper.

---

## Projetos de referência para análise

Antes de escrever qualquer documento, leia e analise os seguintes projetos. Eles são as "peças" que serão combinadas:

### 1. interview-coder-withoupaywall-opensource
**GitHub:** https://github.com/j4wg/interview-coder-withoupaywall-opensource
- Fornece: interface Electron + React invisível ao screen sharing, captura de tela, integração com LLM
- O que aproveitar: estrutura do Electron, `setContentProtection`, atalhos de teclado globais, UI de overlay

### 2. Screenpipe
**GitHub:** https://github.com/mediar-ai/screenpipe
- Fornece: captura de áudio do sistema em macOS (ScreenCaptureKit), Windows e Linux; transcrição automática via Whisper; REST API local na porta 3030 com endpoints de busca em transcrições
- O que aproveitar: daemon local para captura de áudio cross-platform; consumir via `GET http://localhost:3030/search?content_type=audio&limit=N`
- Não precisamos compilar o Screenpipe — ele roda em paralelo como daemon separado

### 3. Meetily
**GitHub:** https://github.com/Zackriya-Solutions/meetily
- Fornece: UX de meeting copilot, separação de locutores (diarização), templates de resumo, integração com Ollama e APIs de LLM
- O que aproveitar: padrões de UI para meeting assistant, prompts de resumo estruturado

### 4. mba-ia-dev-workflow (referência de workflow e documentação)
**Localização local:** `/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/`
- Fornece: o modelo de documentação e processo de desenvolvimento que quero replicar neste projeto
- **Leia obrigatoriamente:** `CLAUDE.md`, `docs/prd/index.md`, `docs/epicos/index.md`, `docs/domain-context.md`, `docs/hld.md`, `docs/roadmap.md`
- Este é o template de como os documentos devem ser estruturados

---

## Stack tecnológica

| Camada | Tecnologia | Justificativa |
|--------|-----------|---------------|
| Interface | Electron + React + TypeScript | Cross-platform; já validado nos projetos de referência; `setContentProtection` nativo |
| Captura de áudio | Screenpipe (daemon externo) | Resolve o problema do macOS ScreenCaptureKit sem reescrever Rust/Swift |
| Transcrição | faster-whisper (Python, CUDA) | 10-20x mais rápido que o Whisper padrão na GPU |
| Backend local | Python (FastAPI ou script simples) | Servidor local para Whisper; ponte entre Screenpipe e Electron |
| LLM | Ollama (local) + APIs cloud | Agnóstico; usuário configura chave/provedor nas settings |
| Armazenamento | Sistema de arquivos local (Markdown + SQLite) | Sem dependência de nuvem para dados pessoais |
| Build | Electron Forge ou electron-builder | Empacotamento multiplataforma |

**TypeScript conventions** (seguir o mba-ia-dev-workflow):
- ES Modules (`"type": "module"`)
- TypeScript strict mode
- Imports com extensão `.js` (ESM requirement)

---

## Workflow de desenvolvimento (OBRIGATÓRIO seguir)

Este projeto segue o mesmo fluxo dos outros projetos do Victor:

```
PRD → FDDs por domínio → Jira Epics + Tasks → /task HF-XX → implementação
```

**Jira:** projeto `HF` — https://goncalvesmarques.atlassian.net (mesmo projeto dos outros serviços)
**Confluence:** documentação de negócio e transversal
**GitHub:** documentação técnica acoplada ao código (AGENTS.md, CLAUDE.md, GEMINI.md, docs/)

### Skills disponíveis após setup
O skill `/task HF-XX` executa o ciclo completo de desenvolvimento — lê o ticket, gera brief para aprovação, implementa, testa e valida o DoD. Funciona em ambas as ferramentas:

| Ferramenta | Diretório de skills | Comando |
|------------|---------------------|---------|
| Claude Code | `.claude/skills/` | `/task HF-XX` |
| agy (Antigravity CLI) | `.agents/skills/` | `/task HF-XX` |

### Prefixos de tipo no título de tickets Jira
Todo ticket deve ter prefixo: `[frontend]`, `[backend]`, `[infra]`, `[observability]`, `[cross-service]`, `[database]`, `[docs]`

### Template de ticket Jira
Seguir exatamente o mesmo template do `mba-ia-dev-workflow/docs/` — com contexto, escopo de arquivos, regras afetadas, critérios de aceite, cenários de teste obrigatórios e evidência (arquivo `.http` ou equivalente para desktop app).

---

## O que quero que você produza nesta sessão

### Etapa 1 — Análise dos projetos de referência
Leia os repositórios GitHub listados acima (via WebFetch ou WebSearch) e o mba-ia-dev-workflow local. Produza um relatório de análise cobrindo:
- Arquitetura de cada projeto (estrutura de pastas, tecnologias, padrões)
- O que pode ser reaproveitado diretamente (fork/import)
- O que precisa ser adaptado
- O que precisa ser construído do zero
- Riscos identificados (ex: dependência de versão do Electron, compatibilidade com macOS)

### Etapa 2 — Perguntas de esclarecimento (se necessário)
Se após a análise houver ambiguidades sobre escopo, prioridade ou decisões técnicas, faça as perguntas antes de produzir os documentos.

### Etapa 3 — PRD
Criar `docs/prd/index.md` seguindo o mesmo formato do mba-ia-dev-workflow:
- Sumário executivo
- Stakeholders (neste caso: só eu, mas com diferentes contextos de uso)
- Requisitos funcionais com prioridade MoSCoW
- Requisitos não funcionais
- Fora do escopo
- Glossário

### Etapa 4 — Domain Context
Criar `docs/domain-context.md` com:
- Entidades do domínio (Meeting, Transcription, Segment, Screenshot, CodeChallenge, LLMProvider, etc.)
- Regras de negócio identificadas

### Etapa 5 — HLD (High-Level Design)
Criar `docs/hld.md` com:
- Diagrama de componentes (Electron main/renderer, Python daemon, Screenpipe daemon, LLM providers)
- Fluxo de dados para cada caso de uso principal
- ADRs iniciais (ex: por que Screenpipe em vez de captura nativa? Por que Python para Whisper?)

### Etapa 6 — Implementation Plan
Criar `docs/implementation-plan.md` com fases, tasks estimadas e dependências — no mesmo formato do hf-income-service.

### Etapa 7 — Jira
Criar no Jira (projeto HF, cloudId `a5d8b89b-37c5-4fd9-b536-5c5f662822fe`):
- 1 Epic por fase do implementation plan
- Tasks para Fase 0 (infraestrutura base: repo, Electron boilerplate, CI)

---

## Restrições e decisões já tomadas

- **Invisibilidade obrigatória:** `win.setContentProtection(true)` no Electron main — não negociável
- **Cross-platform:** nada específico de uma plataforma sem fallback para as outras
- **Screenpipe como daemon externo:** não fork, não bundle — o usuário instala separado; o app detecta se está rodando via `GET http://localhost:3030/health`
- **Dados locais por padrão:** transcrições, atas e screenshots ficam no filesystem local do usuário — zero cloud obrigatório
- **Sem login/auth:** uso pessoal — sem cadastro, sem conta, sem servidor remoto obrigatório
- **LLM keys opcionais:** o app funciona 100% com Ollama local; cloud é opt-in
- **Soft delete:** registros de reuniões nunca deletados fisicamente (apenas marcados como arquivados)

---

## Referências de arquivos locais que você deve ler

```
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/CLAUDE.md
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/docs/prd/index.md
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/docs/domain-context.md
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/docs/hld.md
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/docs/roadmap.md
/home/victor/Projects/personal/home-finance/hf-transaction-service/mba-ia-dev-workflow/docs/epicos/index.md
```

---

## Onde salvar os artefatos

O repositório do novo projeto ainda não existe. Crie a estrutura de pastas em:
`/home/victor/Projects/personal/[PROJECT_NAME]/`

Estrutura mínima esperada ao final da sessão:

```
[PROJECT_NAME]/
├── AGENTS.md              ← instruções cross-tool (Claude Code + agy + Cursor)
├── CLAUDE.md              ← instruções Claude Code-específicas
├── GEMINI.md              ← instruções agy-específicas
├── docs/
│   ├── prd/
│   │   └── index.md
│   ├── domain-context.md
│   ├── hld.md
│   ├── roadmap.md
│   ├── implementation-plan.md
│   ├── workflow.md
│   ├── fdds/
│   └── adr/
├── .claude/
│   └── skills/
│       └── task.md        ← /task para Claude Code
└── .agents/
    └── skills/
        └── task.md        ← /task para agy
```

**Para criar os arquivos de configuração das ferramentas**, copie e adapte os equivalentes do hf-income-service:

| Arquivo a criar | Copiar de |
|----------------|-----------|
| `AGENTS.md` | `/home/victor/Projects/personal/home-finance/hf-income-service/AGENTS.md` |
| `CLAUDE.md` | `/home/victor/Projects/personal/home-finance/hf-income-service/CLAUDE.md` |
| `GEMINI.md` | `/home/victor/Projects/personal/home-finance/hf-income-service/GEMINI.md` |
| `.claude/skills/task.md` | `/home/victor/Projects/personal/home-finance/hf-income-service/.claude/skills/task.md` |
| `.agents/skills/task.md` | `/home/victor/Projects/personal/home-finance/hf-income-service/.agents/skills/task.md` |

Ao adaptar, substitua referências específicas do hf-income-service (domínios REC/ORC/SAL/MET, Go, GORM) pelas tecnologias e domínios deste projeto.

---

## Comece por aqui

1. Leia os arquivos locais do mba-ia-dev-workflow listados acima
2. Leia os arquivos de configuração do hf-income-service listados na tabela acima (para entender o padrão)
3. Faça fetch dos READMEs dos projetos GitHub de referência
4. Produza o relatório de análise (Etapa 1)
5. Pergunte se tiver dúvidas antes de escrever o PRD
