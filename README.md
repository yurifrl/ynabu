# YNABU - Utilitário de Extratos para YNAB

Converte extratos do Itaú para formato CSV compatível com YNAB.

## Instalação
```bash
go install github.com/yurifrl/ynabu/cmd/ynabu@latest
```

## Uso
```bash
# Processar arquivos do diretório atual
ynabu .

# Processar diretório específico
ynabu /dir

# Diretório de saída personalizado
ynabu -o /dir/saida /dir/entrada
```

## Arquivos Suportados
- Extratos bancários Itaú (TXT e XLS)
- Faturas de cartão Itaú (XLS)

Arquivos são salvos como `-ynabu.$EXT.csv` no formato YNAB: Date, Payee, Memo, Amount.

## Desenvolvimento
```bash
# Executar localmente
go run ./cmd/ynabu

# Build
go build -o ynabu ./cmd/ynabu

# Testes
go test ./...