import type { NextConfig } from "next";

// A tela chama sempre o mesmo endereço (/api/...). O servidor do Next repassa esses
// pedidos para a API em Go, então o navegador vê um único site e o cookie de sessão
// funciona sem configuração extra.
const apiInterna = process.env.LG_API_INTERNA ?? "http://127.0.0.1:8090";

const config: NextConfig = {
  transpilePackages: ["@laura-growth/contratos"],
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/api/:caminho*", destination: `${apiInterna}/api/:caminho*` }];
  },
};

export default config;
