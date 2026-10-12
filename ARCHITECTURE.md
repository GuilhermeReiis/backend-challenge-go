# Arquitetura da solução

## Estado atual

O domínio, seus testes, as migrations SQL, a configuração de ambiente, a conexão GORM, os repositories e a Unit of Work estão implementados. A conexão possui hooks Fx para verificação na inicialização e fechamento no encerramento. Services, Echo, autenticação, workers e composição da aplicação ainda serão implementados.

A migration `000006` acrescenta o vínculo jogador/carteira da operação e a validação de direção e resultado na inserção do ledger. Essas garantias foram verificadas por SQL direto em PostgreSQL 17, incluindo aplicação/reversão/reaplicação e recusa de dados existentes incompatíveis. Coerência de saldo/versão/histórico no commit, imutabilidade dos estados terminais e validação completa das referências continuam pendentes.

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

`PayloadHash` de operações externas exige 64 caracteres hexadecimais minúsculos, sem normalização. `NewExternal` e `Rehydrate` compartilham a validação desse formato, compatível com a constraint SQL; formatos inválidos retornam `ErrInvalidPayloadHash`, e metadados obrigatórios vazios continuam retornando `ErrInvalidIdentity`. OPENING interno não possui hash externo. O cálculo canônico do hash será implementado com o comando comum aos services.

O repository adquirirá lock por carteira com `SELECT FOR UPDATE` dentro dessa transação. Não haverá lock global ou garantia baseada apenas na memória do processo. Consultas e escritas receberão `context.Context`. Updates selecionarão explicitamente as colunas, incluindo valores zero e campos que precisem ser limpos.

As migrations SQL continuarão sendo a fonte do schema, incluindo constraints, índices e triggers. A inicialização da aplicação não executará `AutoMigrate`. O ledger manterá somente operações de leitura e inserção. Publicação de eventos ficará no worker da outbox e ocorrerá depois do commit.

Essas garantias de coordenação ainda precisam ser implementadas e demonstradas por testes de integração, concorrência e recuperação.

## Garantias do schema: diagnóstico e plano

Esta seção distingue as garantias implementadas nas migrations `000001` a `000006` dos mecanismos ainda planejados para O02.3 a O02.5. A análise usa os models, mappers, repositories, testes e requisitos das seções 5 a 9 e 11 a 13 do README. Os testes de O02.2 foram executados em bancos temporários; não comprovam as garantias dos blocos seguintes nem aplicam migrations no banco da aplicação.

### Garantias existentes

| Área | Garantia presente no schema | Evidência |
| --- | --- | --- |
| Carteira | Saldo não negativo, versão pelo menos 1, BRL/USD, timestamps ordenados e uma carteira por jogador/moeda | `000001`: checks e `wallets_player_currency_unique` |
| Operação | Origem interna/externa, tipos e estados válidos; LOSS zero e demais tipos positivos; metadados externos obrigatórios e hash hexadecimal minúsculo de 64 caracteres | `000002`: checks de origem, tipo, valor e metadados |
| Identidade externa | Unicidade por provedor/ID externo e por provedor/chave de idempotência | `wager_provider_external_unique`, `wager_provider_idempotency_unique` |
| Abertura e reversão | Uma OPENING por carteira; uma reversão PROCESSED por referência, considerando REFUND e ROLLBACK conjuntamente | `wager_opening_wallet_unique`, `wager_successful_reversal_unique` |
| Referência e resultado | Referência interna existente, diferente do próprio ID, obrigatória quando uma operação referenciada é PROCESSED; resultado não negativo e instante de processamento apenas em PROCESSED | FKs e checks de referência/resultado em `000002` |
| Ledger | Carteira/moeda/valor/operação PROCESSED compatíveis, um lançamento por operação/carteira, valor positivo, saldos não negativos e aritmética exata | FK composta `ledger_transaction_fk`, unicidade e checks em `000003` |
| Jogador da operação | A operação pertence ao jogador e à moeda da carteira; inserções e atualizações incompatíveis são recusadas inclusive por SQL direto | `000006`: `wallets_id_player_currency_unique` e `wager_wallet_player_currency_fk` |
| Direção e resultado do ledger | No INSERT, direção compatível com o tipo/referência e saldo posterior igual ao resultado persistido da operação | `000006`: `ledger_validate_operation`, `validate_ledger_operation` e `ledger_expected_direction` |
| Imutabilidade do ledger | UPDATE, DELETE e TRUNCATE recusados por triggers; aplicação tem somente SELECT/INSERT | `prevent_ledger_mutation` e grants de `000003` |
| Inbox | Identidade consumidor/mensagem única, hash válido e conclusão posterior ao recebimento; aplicação só pode atualizar `completed_at` | `000004` |
| Outbox | Identidade estável e envelope coerente com as colunas, snapshot imutável, campos de reserva coerentes e evento publicado imutável | Checks, `protect_outbox_event` e grants de `000005` |

`LedgerRepository.Append` também verifica direção por tipo e resultado igual ao saldo posterior. `WagerRepository.Create` confere jogador/carteira; `Update` protege identidade e estados terminais. `WalletRepository.Update` exige uma mudança de saldo e versão esperada mais 1. A migration `000006` reforça as verificações de vínculo e INSERT de ledger no banco, sem exigir mudanças nos models/mappers/repositories. As proteções restantes de UPDATE ainda dependem de Go: `wagering_app` tem UPDATE amplo em carteiras e operações.

`000006` roda em transação explícita, com locks de schema nas três tabelas durante a migration, e verifica os dados existentes antes de impor as novas garantias. Jogador incompatível produz SQLSTATE `23503`; direção ou resultado incompatível produz `23514`, com nome da constraint correspondente. Não modifica dados financeiros. O `down` remove somente suas duas constraints, o trigger de INSERT e suas funções; mantém as FKs, checks, grants e triggers originais.

Na inserção do ledger, o trigger adquire primeiro o lock da carteira, depois lê a operação e, para ROLLBACK, a referência com `FOR SHARE`, preservando os dados consultados durante a transação. Ele não confere o saldo atual contra o histórico nem congela resultados contra updates futuros: essas garantias permanecem em O02.3/O02.4. A compatibilidade completa das referências permanece em O02.5.

### Lacunas, mecanismos e testes a implementar

Cada linha associa uma lacuna a um bloco de implementação e a uma prova em PostgreSQL real. Os testes negativos devem usar SQL direto, além dos testes dos repositories, para não depender das validações em Go.

| Lacuna atual | Mecanismo planejado | Bloco | Cenários mínimos |
| --- | --- | --- | --- |
| Saldo pode mudar sem ledger; operação PROCESSED positiva pode existir sem lançamento | Constraint triggers adiados em wallets e wager_transactions exigem correspondência financeira no commit | O02.3 | Falta de ledger em cada lado recusa commit e desfaz todas as escritas; fluxo completo confirma |
| Lançamentos podem partir de saldos inventados e não corresponder à versão da carteira | `wallet_version BIGINT NOT NULL` no ledger, UNIQUE por `(wallet_id, wallet_version)`, trigger de INSERT com lock da carteira e verificação do predecessor | O02.3 | Versão repetida, salto, saldo anterior incorreto e ledger sem atualização correspondente recusados; cadeia válida aceita |
| Banco aceita versão inicial diferente de 1 ou atualização arbitrária de versão | Trigger imediato em wallets exige versão inicial 1; saldo alterado exige versão anterior mais 1; saldo inalterado mantém versão e instante de atualização | O02.3 | Mudança sem incremento, incremento sem movimento, salto, overflow e timestamp regressivo recusados; débito até zero aceito |
| OPENING não está vinculada ao saldo da criação; carteira positiva pode existir sem abertura auditável | Trigger adiado de INSERT da carteira verifica o estado original de criação e OPENING/ledger de versão 1; inserção do ledger valida baseline zero | O02.3 | Abertura positiva incompleta ou divergente recusada; abertura zero sem OPENING aceita; OPENING tardia em carteira criada com zero recusada |
| Resultado de LOSS pode ser um saldo inventado, embora não produza ledger | Ao inserir/finalizar LOSS como PROCESSED, trigger imediato bloqueia a carteira e compara o resultado ao saldo naquele instante | O02.3 | LOSS válido mantém saldo/versão; resultado falso recusado; movimento posterior não altera seu resultado original |
| Identidade e resultado terminal podem ser alterados por SQL; checks só validam o estado final | Triggers BEFORE UPDATE protegem identidade e congelam registros terminais, com comparação que distingue NULL | O02.4 | Mudança de jogador, moeda, valor, chave, hash, referência externa, resultado, failureCode, estado ou timestamps recusada; replay só consulta |
| SQL permite regressão PENDING_REFERENCE para PENDING, referência interna em estado pendente e agenda anterior à atualização | Trigger de transição e checks de forma alinham persistência ao domínio e a `SavePendingReference` | O02.4 | Regressões e agenda inválida recusadas; PENDING → PENDING_REFERENCE → PROCESSED/REJECTED/FAILED e reagendamento válido aceitos |
| FK da referência não confere provedor, ID externo, jogador, carteira, moeda, rodada, tipo, estado ou valor integral | Trigger ao entrar em PROCESSED valida referência e, nas reversões, valor integral; tipos já usados no cálculo da direção permanecem consistentes | O02.5 | Cada incompatibilidade, referência a si mesma, parcial e referência sem sucesso recusadas; WIN referenciado e reversões válidas aceitos |
| Índice já impede duas reversões conjuntas, mas a política e sua resistência a disputas não foram comprovadas | Preservar `wager_successful_reversal_unique`, proteger identidade/estado e validar referências sob lock por carteira | O02.5 | REFUND/REFUND, ROLLBACK/ROLLBACK e REFUND/ROLLBACK sobre a mesma BET: somente uma confirma, inclusive com conexões concorrentes |

### Coordenação e ordem de escrita planejadas

As novas verificações não substituirão o domínio nem executarão movimentações automaticamente. Repositories continuam realizando escritas explícitas; triggers validam as invariantes. Não usar CHECK com consultas a outras tabelas para essas relações.

1. Iniciar a Unit of Work. Para uma carteira existente, adquirir `SELECT FOR UPDATE` antes de alterar inbox/operações, resolver referências ou ler o saldo usado no processamento. Em uma transação excepcional com várias carteiras, adquirir todos os locks em ordem crescente de ID.
2. Registrar/verificar a inbox quando aplicável e reler as identidades da operação. Bloquear operações existentes necessárias em ordem crescente de ID, depois do lock da carteira. Todas as entradas e workers devem seguir essa ordem; colisões de unicidade exigem rollback e releitura em outra transação.
3. Para uma movimentação, aplicar o domínio e atualizar explicitamente saldo/versão com a versão esperada. Persistir a operação PROCESSED e seu resultado original. Inserir imediatamente seu ledger antes da próxima movimentação da mesma carteira.
4. Persistir os snapshots dos eventos correspondentes e concluir a inbox quando aplicável. Solicitar commit; devolver qualquer erro de validação adiada ao chamador, sem publicação externa ou remoção antecipada da mensagem.

Na abertura positiva, a ordem é carteira com saldo inicial e versão 1 → OPENING PROCESSED → ledger de crédito partindo de zero, versão 1 → os dois eventos. Na abertura zero, inserir a carteira com versão 1, sem OPENING, ledger ou eventos financeiros. Uma movimentação posterior na carteira aberta com zero começa no ledger de versão 2, com saldo anterior zero.

LOSS, REJECTED, FAILED e PENDING_REFERENCE não atualizam saldo/versão nem inserem ledger. LOSS preserva o saldo observado sob lock e produz somente o evento de processamento. Rejeições e pendências confirmam os eventos correspondentes; uma rejeição auditável retorna sucesso do callback SQL e seu resultado de negócio fora dele. Não definir neste bloco aceite assíncrono de PENDING ou política de falhas permanentes: essas decisões pertencem a O06/O10.

Os triggers que consultam saldo/histórico também adquirirão o lock da carteira para proteger SQL direto. Isso não corrige uma ordem de locks invertida pelo chamador: services/repositories devem respeitar a ordem acima; deadlocks/erros de serialização continuam sujeitos ao retry limitado de O07. Testar duas conexões na mesma carteira e progresso independente em carteiras distintas.

### Validação da cadeia e do commit planejada

`wallet_version` é metadado da persistência; não será uma nova responsabilidade da entidade `ledger.Entry`. `LedgerRepository.Append` obterá a versão da carteira bloqueada e preencherá o model explicitamente. O trigger verificará a versão recebida. Models/mappers serão ajustados apenas para carregar esse metadado, mantendo os contratos de domínio.

O INSERT do ledger deve conferir o saldo/versão atuais da carteira, o resultado da operação e o predecessor na mesma carteira. OPENING usa versão 1 e saldo anterior zero; o primeiro movimento sem OPENING usa versão 2 e saldo anterior zero. Os demais usam a versão imediatamente anterior e seu saldo posterior. `sequence` continua sendo o cursor estável de leitura; não representa a versão financeira nem exige números globais contíguos. A unicidade por carteira/versão fornece o índice para verificar a cadeia sem somar todo o histórico a cada escrita.

Um trigger adiado por atualização financeira da carteira deve preservar os valores OLD/NEW daquela atualização e exigir o ledger da respectiva versão, com saldos anterior/posterior correspondentes. Não comparar todo lançamento intermediário apenas com o saldo final da carteira. Um trigger adiado por operação exige exatamente um lançamento quando o estado final é PROCESSED e o tipo movimenta dinheiro; LOSS e estados sem sucesso não admitem ledger, mantendo também as FKs existentes. A criação da carteira deve ser conferida pelo seu snapshot de INSERT, mesmo quando ocorrerem movimentos posteriores na mesma transação SQL.

As validações finais também conferem que a carteira corresponde ao último lançamento de sua versão, ou ao estado zero/versão 1 sem lançamentos. A cadeia é validada na inserção, com locks, e as ligações faltantes são recusadas no commit. Isso impede, por exemplo, crédito e débito compensatórios na mesma transação de esconder uma versão sem ledger. Os cálculos intermediários SQL usam NUMERIC exato para detectar overflow antes de persistir BIGINT; não introduzir ponto flutuante.

Constraint triggers serão `AFTER ROW DEFERRABLE INITIALLY DEFERRED`. Os testes devem observar falhas efetivas no commit, verificar rollback e conferir também a antecipação por `SET CONSTRAINTS ... IMMEDIATE`. Guards de identidade, transição, versão e INSERT de ledger permanecem imediatos. Uma operação financeira parcialmente escrita dentro da Unit of Work é transitória; não pode sobreviver ao commit.

### Identidade e estados planejados

O02.4 protegerá `id`, `player_id`, `currency` e `created_at` da carteira. Nas operações, protegerá os campos comparados por `sameWagerIdentity`: ID, origem, tipo, carteira, jogador, moeda, valor, criação, provedor, ID externo, chave, hash, rodada, jogo e referência externa.

PENDING pode permanecer PENDING ou ir para PENDING_REFERENCE, PROCESSED, REJECTED ou FAILED. PENDING_REFERENCE pode permanecer nesse estado para reagendamento ou ir para um dos três estados terminais; não volta a PENDING. A referência interna será resolvida no processamento; os estados pendentes mantêm esse campo NULL. A agenda de PENDING_REFERENCE exige `next_attempt_at >= updated_at` e expiração posterior à criação, conforme a validação atual do repository.

Uma linha terminal será imutável por completo, inclusive resultado, motivo, referência interna, agenda e timestamps. UPDATE sem mudança de valores pode ser aceito; replay não usa UPDATE. Comparações de campos anuláveis usarão `IS DISTINCT FROM`. A aplicação continuará sem permissões de DELETE/TRUNCATE dessas tabelas e sem privilégios de proprietário ou de desativação dos triggers. Essas proteções não pretendem impedir um administrador de alterar deliberadamente o schema.

### Política de referências e reversões planejada

Ao processar uma referência, exigir operação EXTERNAL/PROCESSED, concordância de provedor, jogador, carteira, moeda e rodada, e correspondência entre referência externa e ID interno resolvido. WIN referenciado exige BET; seu valor não precisa coincidir com o da aposta. REFUND exige BET; ROLLBACK exige BET, WIN ou REFUND e movimenta na direção contrária. REFUND/ROLLBACK exigem valor integral. Referência ausente ou ainda pendente deve seguir O08/O10; referência terminal sem sucesso não pode produzir uma reversão PROCESSED. O service continua responsável por persistir a rejeição ou pendência e seus códigos/eventos.

Manter a política já representada pelo índice atual: uma operação original admite no máximo uma reversão bem-sucedida, somando REFUND e ROLLBACK. Assim, uma BET já devolvida por REFUND não aceita outro REFUND nem ROLLBACK direto sobre a BET. É permitido ROLLBACK sobre o próprio REFUND, debitando a devolução se houver saldo; isso não libera a BET para outra devolução. Também não se permite ROLLBACK de ROLLBACK. Tentativas rejeitadas não ocupam a unicidade de reversão bem-sucedida; saldo insuficiente numa reversão com débito será auditado pelo service com código distinto de BET.

O índice garante a exclusão de reversões concorrentes; locks e validação da referência garantem coerência. Não remover o índice nem trocá-lo por unicidade que inclua o tipo de reversão, pois isso permitiria devolver a mesma BET uma vez por REFUND e outra por ROLLBACK.

### Migrations e compatibilidade dos testes

O02.3 a O02.5 acrescentarão migrations `up`/`down`; não alterar as migrations `000001` a `000006`. Antes de impor novas constraints ou preencher `wallet_version`, verificar os dados existentes: cadeia, saldo final, versão e referências. Recusar uma migration sobre dados inconsistentes, indicando a lacuna, sem corrigir saldo/histórico silenciosamente.

Se houver dados válidos, derivar as versões por carteira na ordem de `sequence`, considerando a presença de OPENING. O preenchimento é uma operação administrativa da migration, sob lock exclusivo e na mesma transação SQL da mudança de schema. Como o ledger já recusa UPDATE inclusive administrativo, qualquer suspensão estritamente necessária desse trigger deve permanecer dentro dessa transação bloqueada, com restauração antes do commit; não criar uma exceção utilizável pela aplicação. O `down` remove apenas os novos mecanismos/metadados, sem apagar ou recalcular movimentos e sem retirar a proteção original do ledger.

Os testes atuais em `repositories_integration_test.go` precisam de ajustes no bloco que acrescentar a garantia correspondente:

- O teste `persists zero balance and rejects stale versions` atualmente confirma um débito sem operação/ledger. Em O02.3, manter a prova de saldo zero e versão obsoleta usando um débito completo, e acrescentar a recusa do fluxo incompleto.
- Os testes de agenda, duplicidade e locks criam carteiras positivas sem OPENING/ledger. Em O02.3, usar abertura completa ou saldo inicial zero conforme o objetivo do caso.
- A fixture `persistOpening` grava só `WagerTransactionProcessed`; em O03, completar também `WalletBalanceChanged` para provar o contrato de abertura. O teste atual não comprova a presença dos dois eventos obrigatórios.
- A prova de estado terminal atual passa por `WagerRepository.Update`; em O02.4, acrescentar SQL direto sobre resultado/estado/identidade.
- As fixtures de `ledger_integrity_integration_test.go` isolam as garantias de cada lançamento de O02.2; não representam uma cadeia financeira completa. Em O02.3, adaptá-las à nova exigência de saldo/versão/ledger, mantendo os testes negativos independentes das validações dos repositories.
- Em cada bloco, verificar novas migrations em banco isolado vazio e com dados válidos, rejeição de dados incompatíveis quando aplicável, `down`/`up` e preservação das proteções anteriores. Reaproveitar o banco temporário e o papel `wagering_app` do helper de integração.

Os testes de `ledger_integrity_integration_test.go` comprovam vínculo de jogador na inserção/atualização, as sete direções válidas e seus opostos, resultado divergente, referências sem direção de reversão válida, preservação de constraints anteriores e rollback diante de ledger inválido. Também comprovam `up` em banco vazio, `down`/`up` sobre dados válidos, recusa de dados incompatíveis sem correção ou schema parcialmente aplicado e proteção original contra UPDATE/DELETE/TRUNCATE antes/depois da reaplicação.

Foram executados `go test ./...`, `go test -tags=integration ./internal/repository/postgres/... -count=1`, `go vet ./...` e `go vet -tags=integration ./internal/repository/postgres/...`. Os cenários de O02.3 a O02.5 continuam definidos, sem implementação ou prova; testes atuais podem passar enquanto essas lacunas permanecem abertas. O detector de race continua no contexto próprio de preparação do ambiente.

### Lacunas que dependem de outros contextos

Inbox/outbox já possuem unicidade e proteção de formato/snapshot, mas o schema não exige eventos para cada operação nem vincula a conclusão da inbox ao comando financeiro. Essa coordenação será implementada nos mesmos callbacks da Unit of Work pelos services (O03, O06/O07/O09/O10) e consumidor (O16), com testes de omissão/falha de escrita e contagem dos eventos em O20. O02 não implementará services, publicação ou retomada de workers.

Replay com comparação de hash/ID externo e resultado original pertence a O05/O06; a imutabilidade planejada em O02.4 protege os dados usados por esse fluxo. Reservas e recuperação de inbox/outbox e referências permanecem em O10/O16/O17. Isolamento de provedor e autorização continuam em O04/O12/O13/O16; FKs de jogador e referência não autenticam o chamador.

As APIs atuais de ledger permitem construir novos lançamentos válidos, mas não comprovam reconciliação consistente ou ausência de atualizações perdidas entre processos. O11 fará a reconciliação e O21/O22 comprovarão concorrência e recuperação com múltiplos processos.

Referências técnicas usadas para o plano: [constraints no PostgreSQL 17](https://www.postgresql.org/docs/17/ddl-constraints.html), [constraint triggers e adiamento](https://www.postgresql.org/docs/17/sql-createtrigger.html) e [locks de linha e ordem de aquisição](https://www.postgresql.org/docs/17/explicit-locking.html). As provas da implementação continuarão sendo os testes do projeto em PostgreSQL real.

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
