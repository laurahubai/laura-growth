import { NextResponse, type NextRequest } from "next/server";

// Sem cookie de sessão, qualquer tela leva para /entrar. A validade da sessão
// é conferida pela API; aqui é só um atalho para não piscar a tela.
export function proxy(request: NextRequest) {
  const temSessao = Boolean(request.cookies.get("lg_sessao")?.value);
  const naEntrada = request.nextUrl.pathname.startsWith("/entrar");
  if (!temSessao && !naEntrada) {
    return NextResponse.redirect(new URL("/entrar", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!api/|_next/|favicon\\.ico$).*)"],
};
