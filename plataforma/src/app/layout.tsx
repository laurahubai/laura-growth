import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { LojaProvider } from "@/mock/loja";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

export const metadata: Metadata = { title: "Laura Growth", description: "Seu time de marketing com IA (protótipo com dados de exemplo)" };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR" className={inter.variable}>
      <body>
        <LojaProvider>{children}</LojaProvider>
      </body>
    </html>
  );
}
