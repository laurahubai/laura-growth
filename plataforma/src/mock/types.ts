// Tipos do domínio. Ficam iguais quando o backend real substituir o mock.

export type Rede = "instagram" | "facebook" | "linkedin";
export type Formato = "imagem" | "carrossel";
export type Etapa = 1 | 2 | 3 | 4 | 5 | 6;
export type Agente = "Strategy Agent" | "Creative Agent" | "Social Agent";

export type EstadoPost =
  | "em_criacao"
  | "aguardando_aprovacao"
  | "agendado"
  | "pronto_manual"
  | "publicado"
  | "falhou"
  | "atrasado"
  | "descartado";

export interface KitMarca {
  cores: string[];
  fontes: string;
  tomDeVoz: string;
  temas: string;
  publico: string;
  nuncaDizer: string;
  exemplos: string;
}

export interface Organizacao {
  id: string;
  nome: string;
  tipo: "empresa" | "clinica_estetica";
  kit: KitMarca;
  cadencia: Record<Rede, number>; // posts por semana
  conexoes: Record<Rede, "nao_conectada" | "ativa" | "expirada">;
}

export interface Pauta {
  tema: string;
  objetivo: string;
  redes: Rede[];
  formato: Formato;
}
export interface Texto {
  legenda: string;
  hashtags: string;
}
export interface Slide {
  titulo: string;
  imagem?: string; // imagem enviada pela pessoa (dataURL)
}
export interface Visual {
  slides: Slide[];
}
export interface Revisao {
  avisos: string[];
}

export interface Versao {
  id: string;
  etapa: Etapa;
  autor: "ia" | "pessoa";
  comentario?: string;
  em: string;
  conteudo: Pauta | Texto | Visual | Revisao;
}

export interface Evento {
  em: string;
  quem: string;
  acao: string;
}

export interface Post {
  id: string;
  orgId: string;
  origem: "sugestao" | "pedido";
  etapa: Etapa;
  data: string; // aaaa-mm-dd prevista
  descartado: boolean;
  publicacao?: "publicado" | "falhou";
  aprovadoPor?: string;
  pauta: Pauta;
  texto?: Texto;
  visual?: Visual;
  revisao?: Revisao;
  versoes: Versao[];
  historico: Evento[];
  atividade: { em: string; texto: string }[];
  trabalhando: boolean;
}

export interface RegistroCusto {
  id: string;
  orgId: string;
  postId: string;
  agente: Agente;
  etapa: Etapa;
  fornecedor: string;
  unidade: "tokens" | "imagem";
  quantidade: number;
  precoUnitarioUsd: number; // preço guardado no registro (CST-07)
  custoUsd: number;
  em: string;
}

export interface Dados {
  versao: number;
  orgs: Organizacao[];
  posts: Post[];
  custos: RegistroCusto[];
}
