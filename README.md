# ZIP Password Cracker

Este é um aplicativo web que permite quebrar senhas de arquivos ZIP protegidos usando uma lista de palavras (wordlist). O aplicativo usa WebSockets para fornecer atualizações em tempo real sobre o progresso do processo de quebra de senha.

## Sobre o Projeto

O ZIP Password Cracker é uma ferramenta desenvolvida em Go que utiliza:
- **Gin**: Framework web para manipulação de requisições HTTP
- **WebSockets**: Para comunicação em tempo real com o cliente
- **alexmullins/zip**: Para manipulação de arquivos ZIP protegidos por senha

O projeto foi desenvolvido como parte de uma atividade acadêmica para demonstrar conceitos de segurança e quebra de senhas.

## Recursos

- Interface web intuitiva
- Upload de arquivos ZIP e wordlists
- Progresso em tempo real do processo de quebra de senha
- Exibição das últimas senhas testadas
- Notificação imediata quando a senha correta é encontrada

## Requisitos

- Go 1.18 ou superior
- Git

## Instalação

Siga estes passos para instalar e configurar o projeto:

1. Clone o repositório:
```bash
git clone https://github.com/seu-usuario/zip-password-cracker.git
```

2. Navegue até o diretório do projeto:
```bash
cd zip-password-cracker
```

3. Instale as dependências:
```bash
go mod download
```

4. Execute o aplicativo:
```bash
go run main.go
```

## Uso

1. Abra o navegador e acesse `http://localhost:8080`.
2. Faça o upload do arquivo ZIP e da wordlist.
3. Acompanhe o progresso em tempo real e aguarde até que a senha seja encontrada.

## Licença

Este projeto está licenciado sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.
