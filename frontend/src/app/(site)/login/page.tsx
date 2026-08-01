"use client";

import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useUser } from "@/lib/user";
import { useI18n } from "@/lib/i18n/client";
import PhoneLogin from "@/components/auth/PhoneLogin";

function LoginCard() {
  const router = useRouter();
  const params = useSearchParams();
  const { user, loading } = useUser();
  const { t } = useI18n();

  // Where to go after signing in — set by the cart when an order is pending.
  const next = params.get("next") || "/profile";
  const reason = params.get("reason");

  useEffect(() => {
    if (!loading && user) router.replace(next);
  }, [user, loading, router, next]);

  return (
    <main className="container-page flex min-h-[60vh] items-center justify-center py-12">
      <div className="card w-full max-w-sm p-8 text-center">
        <h1 className="font-display text-2xl font-bold">{t.login.title}</h1>
        <p className="mt-2 text-sm text-ink-muted">
          {reason === "order"
            ? t.login.authRequired
            : reason === "booking"
              ? t.login.bookingRequired
              : t.login.text}
        </p>

        <div className="mt-6">
          <PhoneLogin onSuccess={() => router.replace(next)} />
        </div>
      </div>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginCard />
    </Suspense>
  );
}
