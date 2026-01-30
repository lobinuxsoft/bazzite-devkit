---
name: pr
description: Create Pull Requests following project workflow.
---
# Pull Request Workflow (CapyDeploy)

## Branch Structure
```
master (releases) <- feature/issue-ID-desc
```

## Flow
1. **Issue First**: Siempre crear issue antes de trabajar
2. **Branch**: Crear desde issue con `gh issue develop`
3. **Work**: Commits en espanol, Conventional Commits
4. **PR**: Crear PR hacia `master`
5. **STOP**: Esperar instrucciones del usuario (NO continuar al siguiente issue)
6. **Close**: PR cierra el issue automaticamente

## GitHub CLI Commands

### Crear branch desde issue
```bash
gh issue develop <NUM> --base master --checkout
```

### Crear Pull Request
```bash
gh pr create --base master --title "feat: descripcion" --body "Closes #XX"
```

### Agregar labels a issue
```bash
gh issue edit <NUM> --add-label "priority:high"
gh issue edit <NUM> --add-label "next-session"
```

## Labels
- `priority:low|medium|high|critical`
- `difficulty:easy|medium|hard`
- `next-session` - Para retomar en proxima sesion
- `platform:windows|linux` - Especifico de plataforma
- `backend|frontend` - Area del codigo

## PR Template
```markdown
## Resumen
Breve descripcion de los cambios.

## Cambios
- Cambio 1
- Cambio 2

## Testing
- [ ] Compilado en Windows
- [ ] Compilado en Linux
- [ ] Tests pasan
- [ ] Probado manualmente

Closes #XX
```

## SemVer (para releases)
- **MAJOR** (vX.0.0): Breaking changes, incompatibilidad config
- **MINOR** (v0.X.0): Nueva funcionalidad
- **PATCH** (v0.0.X): Bug fixes

## Rules
- NUNCA force push a `master`
- PRs van a `master`
- Despues de crear PR: STOP y esperar al usuario
- NUNCA hacer merge a menos que el usuario lo pida explicitamente
- Probar en ambas plataformas antes de merge (cuando sea posible)
- **NO incluir firmas de AI** en el body del PR (ej: "Generated with Claude")
