# GEMINI.md

> Instruções específicas para o Antigravity CLI (agy).
> Leia também `AGENTS.md` — contém todas as instruções de arquitetura, DoD e convenções do projeto.

## Prioridade de carregamento (agy)

```
GEMINI.md  → instruções Antigravity-específicas (este arquivo)
AGENTS.md  → instruções cross-tool (arquitetura, DoD, regras)
```

## Skills disponíveis

As skills estão em `.agents/skills/` e viram slash commands no agy:

| Comando | Arquivo | O que faz |
|---------|---------|-----------|
| `/task HF-XX` | `.agents/skills/task.md` | Ciclo completo de desenvolvimento a partir de um ticket Jira |
| `/fdd-creator <DOMINIO>` | `.agents/skills/fdd-creator.md` | Entrevista estruturada para gerar FDD |
| `/implementation-plan-creator` | `.agents/skills/implementation-plan-creator.md` | Gera plano de implementação a partir dos FDDs |

## MCP Servers

Os mesmos MCP servers configurados localmente estão disponíveis:
- **Atlassian MCP** — Jira (projeto HF) e Confluence
- cloudId Jira: `a5d8b89b-37c5-4fd9-b536-5c5f662822fe`

## Modelo recomendado

- Tasks rotineiras (`/task`): Gemini 3.5 Flash (High)
- FDD e planejamento (`/fdd-creator`, `/implementation-plan-creator`): modelo mais capaz disponível

## Comportamento preferido

- Confirmar antes de criar tickets Jira em lote (mais de 5 de uma vez)
- Nunca iniciar implementação antes do CHECKPOINT 1 do `/task` ser aprovado
- Nunca commitar sem instrução explícita do usuário
- Ao usar `/yolo`, confirmar escopo antes de ativar
