import React from 'react';
import { Navbar } from './components/Navbar';
import { HeroTerminal } from './components/HeroTerminal';
import { Architecture } from './components/Architecture';
import { Benchmarks } from './components/Benchmarks';
import { Features } from './components/Features';
import { ThemePreview } from './components/ThemePreview';
import { InstallSection } from './components/InstallSection';
import { Footer } from './components/Footer';

export const App: React.FC = () => {
  return (
    <div className="min-h-screen flex flex-col bg-[#1a1b26] text-[#c0caf5]">
      <Navbar />
      <main className="flex-1">
        <HeroTerminal />
        <Architecture />
        <Benchmarks />
        <Features />
        <ThemePreview />
        <InstallSection />
      </main>
      <Footer />
    </div>
  );
};

export default App;
