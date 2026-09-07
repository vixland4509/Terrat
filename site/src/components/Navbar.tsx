import React, { useState } from 'react';
import { Terminal, Sparkles, Copy, Check } from 'lucide-react';
import { GithubIcon } from './GithubIcon';

export const Navbar: React.FC = () => {
  const [copied, setCopied] = useState(false);

  const handleCopyInstall = () => {
    navigator.clipboard.writeText('go install github.com/vixland4509/Terrat@latest');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <nav className="sticky top-0 z-50 bg-[#13141c]/90 backdrop-blur-md hairline-b">
      <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
        {/* Logo & Brand */}
        <div className="flex items-center gap-3">
          <img src="./icon.png" alt="Terrat Logo" className="w-8 h-8 rounded-full border border-[#292e42]" />
          <span className="font-mono font-bold tracking-wider text-white text-lg">TERRAT</span>
        </div>

        {/* Navigation Links */}
        <div className="hidden md:flex items-center gap-8 font-mono text-xs uppercase tracking-wider text-[#565f89]">
          <a href="#architecture" className="hover:text-[#c0caf5] transition-colors">Architecture</a>
          <a href="#benchmarks" className="hover:text-[#c0caf5] transition-colors">Benchmarks</a>
          <a href="#features" className="hover:text-[#c0caf5] transition-colors">Features</a>
          <a href="#themes" className="hover:text-[#c0caf5] transition-colors">Themes</a>
          <a href="#install" className="hover:text-[#c0caf5] transition-colors">Install</a>
        </div>

        {/* Right CTA */}
        <div className="flex items-center gap-3">
          <button
            onClick={handleCopyInstall}
            className="hidden sm:flex items-center gap-2 font-mono text-xs px-3 py-1.5 rounded-[2px] bg-[#1a1b26] hover:bg-[#24283b] border border-[#292e42] text-[#c0caf5] transition-all"
            title="Copy install command"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-[#9ece6a]" /> : <Copy className="w-3.5 h-3.5 text-[#7aa2f7]" />}
            <span>go install terrat</span>
          </button>

          <a
            href="https://github.com/vixland4509/Terrat"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 font-mono text-xs px-3 py-1.5 rounded-[2px] bg-[#7aa2f7] hover:bg-[#89b4fa] text-[#13141c] font-semibold transition-all"
          >
            <GithubIcon className="w-4 h-4" />
            <span className="hidden sm:inline">GitHub</span>
          </a>
        </div>
      </div>
    </nav>
  );
};
