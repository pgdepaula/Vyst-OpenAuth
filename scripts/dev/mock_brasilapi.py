import sys
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

port = int(sys.argv[1]) if len(sys.argv) > 1 else 8983

class MockHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/api/cnpj/v1/"):
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            response = {
                "cnpj": "00000000000191",
                "razao_social": "MOCK COMPANY LTDA",
                "descricao_situacao_cadastral": "ATIVA",
                "nome_fantasia": "MOCK FANTASIA",
                "cnae_fiscal": 1234567,
                "natureza_juridica": "Sociedade Empresária Limitada",
                "logradouro": "RUA FAKE",
                "numero": "123",
                "bairro": "CENTRO",
                "municipio": "SÃO PAULO",
                "uf": "SP",
                "cep": "01000000"
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == "__main__":
    server = HTTPServer(("localhost", port), MockHandler)
    server.serve_forever()
