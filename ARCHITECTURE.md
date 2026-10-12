# Arquitetura da solução

## Estado atual

O domínio, seus testes, as migrations SQL, a configuração de ambiente, a conexão GORM, os repositories e a Unit of Work estão implementados. A conexão possui hooks Fx para verificação na inicialização e fechamento no encerramento. Services, Echo, autenticação, workers e composição da aplicação ainda serão implementados.

## Organização do domínio

Todos os arquivos de cada diretório pertencem ao mesmo pacote Go. A separação em arquivos organiza responsabilidades; não acrescenta camadas ou dependências.

| Pacote | Responsabilidade | Organização |
| --- | --- | --- |
| `money` | Valor monetário imutável e operações exatas | `money.go`, `constructor.go`, `serialization.go`, `errors.go` |
| `wallet` | Estado da carteira e débito/crédito | `wallet.go`, `constructor.go`, `rehydrate.go`, `validation.go`, `errors.go` |
| `wager` | Operação financeira e máquina de estados | `transaction.go`, `constructor.go`, `rehydrate.go`, `validation.go`, `types.go`, `errors.go` |
| `ledger` | Lançamento imutável e consistência dos saldos | `entry.go`, `constructor.go`, `types.go`, `errors.go` |
| `event` | Envelope e snapshots tipados | Envelope compartilhado e um arquivo por tipo de evento |

Os testes permanecem no pacote externo correspondente, em arquivos `_test.go`, com subtestes e tabelas.

`New` cria uma entidade; `Rehydrate` valida e restaura seu estado persistido, sem executar novas movimentações ou emitir eventos. O repository será responsável pela consulta e chamará a reidratação para devolver uma entidade válida.

## Camadas da aplicação

```text
cmd/server/                    ponto de entrada
internal/config/               configuração e validação de ambiente
internal/domain/               entidades, valores, regras e eventos
internal/service/              operações da aplicação e contratos de persistência
internal/repository/postgres/  GORM, models, mappers e transação SQL
internal/transport/http/       Echo, handlers, DTOs e middleware
internal/transport/sqs/        consumidor e publicação
internal/worker/               outbox e referências pendentes
internal/bootstrap/            composição e ciclo de vida com Fx
migrations/                    evolução explícita do schema
deploy/                        preparação do ambiente local
```

Os diretórios das camadas ainda não implementadas serão adicionados junto com seu código.

HTTP e SQS chamarão os mesmos services para aplicar as mesmas regras. Handlers converterão o contrato HTTP em entradas do service e seus resultados em respostas. Services usarão o domínio e interfaces de persistência. Repositories implementarão essas interfaces com GORM. O domínio não importará Echo, GORM, Fx ou SQS.

Os contratos estão em `internal/service/repositories.go`. O contrato de transação está em `internal/service/unit_of_work.go`. A implementação PostgreSQL fornecerá os repositories vinculados à mesma transação, sem expor `*gorm.DB` ao service. Fx fará a ligação entre contratos e implementações no bootstrap.

`UnitOfWork.Execute` recebe um callback com os repositories da transação. Retornar `nil` solicita commit; retornar um erro solicita rollback. Falhas no commit são devolvidas ao chamador. Os repositories não podem ser reutilizados depois do callback nem compartilhados entre goroutines. Escritas e leitura com lock exigem uma transação; consultas simples poderão usar repositories fora dela.

Os repositories devolverão `ErrNotFound`, `ErrAlreadyExists`, `ErrConcurrentUpdate` e `ErrTransactionRequired`, quando aplicável, preservando a classificação por `errors.Is`. Erros de infraestrutura permanecerão distintos. Uma colisão de unicidade não significa automaticamente replay válido: o service ainda deverá comparar a identidade e o hash com o registro persistido.

O update da carteira recebe a versão esperada para detectar alterações concorrentes. A leitura com lock e o update permanecem na mesma transação. A agenda de uma referência pendente é persistida junto com seu estado por `SavePendingReference`. A consulta do ledger usa sequência crescente; o transporte converterá o cursor em um token opaco. A outbox recebe o snapshot serializado de um evento tipado, com metadados correspondentes.

Os contratos desta etapa cobrem consultas e gravação do tratamento financeiro. Reserva de eventos e retomada de pendências pelos workers serão adicionadas junto com suas implementações.

As implementações em `internal/repository/postgres` estão separadas nas pastas `model`, `mapper` e `repository`, com a conexão na raiz. Models e funções de mapeamento são exportados entre esses pacotes. A Unit of Work abre uma transação GORM e fornece todos os repositories vinculados a ela. Ao concluir o callback, esses repositories são invalidados. Constructors fora da Unit of Work permitem consultas simples; escritas e locks devolvem `ErrTransactionRequired`.

Os updates de carteira usam versão esperada e colunas explícitas. Updates de transação bloqueiam o registro, preservam os campos de identidade e recusam registros terminais. Inserções conferem a identidade do jogador da carteira. O ledger valida sua relação com a operação processada, valor, saldo resultante e direção antes da inserção; o schema continua impondo suas constraints e imutabilidade. A outbox confere os metadados contra o envelope serializado, sem decodificar valores monetários em ponto flutuante.

Essas verificações dos repositories não substituem as regras dos services nem completam as garantias financeiras do desafio. A coerência entre referências, a movimentação acompanhada de ledger e eventos, replay, autorização e recuperação ainda precisam dos services e workers e de suas verificações no banco.

`TIMESTAMPTZ` usa precisão de microssegundos: os mappers normalizam timestamps de entidades para essa precisão. O payload da outbox preserva seu timestamp original; sua coluna `occurred_at` usa o arredondamento correspondente ao cast PostgreSQL do timestamp do JSON para manter a constraint de metadados consistente.

## Persistência e transação financeira

Models GORM representarão as colunas do schema. Mappers converterão models em entidades por meio da reidratação, preservando encapsulamento e validações. DTOs HTTP serão separados dos models e das entidades.

`Money` usa `int64` em unidades mínimas e moeda explícita, com escala de duas casas. O banco usa `BIGINT`; o contrato externo usa strings decimais. O intervalo interno é o de `int64`, e parsing e operações detectam valores fora dos limites. Saldos e entradas externas não aceitam negativos; diferenças internas podem ser negativas.

Uma operação financeira deverá confirmar estado da transação, saldo, ledger, inbox quando aplicável e outbox na mesma transação SQL. A Unit of Work delimitará esse commit. Um resultado de rejeição de negócio deverá ser persistido e confirmado; uma falha de infraestrutura durante o tratamento deverá provocar rollback.

O repository adquirirá lock por carteira com `SELECT FOR UPDATE` dentro dessa transação. Não haverá lock global ou garantia baseada apenas na memória do processo. Consultas e escritas receberão `context.Context`. Updates selecionarão explicitamente as colunas, incluindo valores zero e campos que precisem ser limpos.

As migrations SQL continuarão sendo a fonte do schema, incluindo constraints, índices e triggers. A inicialização da aplicação não executará `AutoMigrate`. O ledger manterá somente operações de leitura e inserção. Publicação de eventos ficará no worker da outbox e ocorrerá depois do commit.

Essas garantias de coordenação ainda precisam ser implementadas e demonstradas por testes de integração, concorrência e recuperação.

## Convenções de código

- Construtores recebem dependências explicitamente; não há conexão global.
- Services coordenam operações; regras das entidades permanecem no domínio.
- Interfaces expressam operações específicas, evitando um repository CRUD genérico.
- Erros são classificáveis; conversão em status HTTP pertence ao transporte.
- Blocos `if` usam corpo em múltiplas linhas e são separados dos comandos anteriores e seguintes por uma linha em branco, quando houver outros comandos no mesmo bloco.
- Chamadas cujo erro é tratado imediatamente e cujas variáveis só são necessárias no próprio `if` usam inicialização na condição, como `if err := t.validateTransition(now); err != nil {`. Quando outros valores retornados são necessários depois do `if`, eles permanecem no escopo adequado. Erros são tratados com retorno antecipado.
- Helpers de validação pertencem ao pacote da entidade e são compartilhados quando a regra é a mesma.
- Model, mapper e repository de cada entidade ficam próximos no pacote PostgreSQL.
- Novos arquivos são criados quando existe uma responsabilidade concreta a implementar.
