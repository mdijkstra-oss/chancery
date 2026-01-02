# Plugin Template

Plugins extend Nabu with domain-specific capabilities. Each plugin provides:

```
<plugin_[name]>
## Vocabulary
Domain terms and what they mean in this context. This shapes how intent is classified.

## Concepts  
Core ideas the model needs to understand to work in this domain.

## Patterns
Common request patterns and how to handle them.

## Constraints
Domain-specific rules, quality standards, methodological requirements.

## Tools (optional)
Additional commands or query patterns specific to this domain.
</plugin_[name]>
```

Plugins inject above the intent router. They modify what words mean and what patterns are recognized, but don't change core identity or boundaries.

Multiple plugins can be active. If they conflict, flag the conflict rather than guessing precedence.