"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Home,
  Footprints,
  Moon,
  Heart,
  Dumbbell,
  Scale,
  Utensils,
} from "lucide-react";
import { cn } from "@/lib/utils";

const navItems = [
  { href: "/", label: "홈", icon: Home },
  { href: "/steps", label: "걸음수", icon: Footprints },
  { href: "/sleep", label: "수면", icon: Moon },
  { href: "/heart-rate", label: "심박수", icon: Heart },
  { href: "/exercise", label: "운동", icon: Dumbbell },
  { href: "/body", label: "체성분", icon: Scale },
  { href: "/nutrition", label: "영양", icon: Utensils },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:flex md:w-56 md:flex-col md:fixed md:inset-y-0 border-r border-border bg-card">
        <div className="px-4 py-6">
          <h1 className="text-lg font-bold tracking-tight">Health Hub</h1>
          <p className="text-xs text-muted-foreground">건강 대시보드</p>
        </div>
        <nav className="flex-1 px-2 space-y-1">
          {navItems.map((item) => {
            const active =
              item.href === "/"
                ? pathname === "/"
                : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground",
                )}
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </aside>

      {/* Mobile bottom tabs */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 z-50 border-t border-border bg-card">
        <div className="flex justify-around py-2">
          {navItems.slice(0, 5).map((item) => {
            const active =
              item.href === "/"
                ? pathname === "/"
                : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex flex-col items-center gap-0.5 px-2 py-1 text-xs transition-colors",
                  active
                    ? "text-foreground"
                    : "text-muted-foreground",
                )}
              >
                <item.icon className="h-5 w-5" />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </div>
      </nav>
    </>
  );
}
