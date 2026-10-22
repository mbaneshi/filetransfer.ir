"use client";

import { useState } from "react";
import { api, errorMessage, toEnglishDigits } from "../lib/api";

export default function LoginPage() {
  const [step, setStep] = useState<"phone" | "code">("phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      if (step === "phone") {
        await api("/auth/otp/request", { method: "POST", json: { phone: toEnglishDigits(phone) } });
        setStep("code");
      } else {
        await api("/auth/otp/verify", { method: "POST", json: { phone: toEnglishDigits(phone), code: toEnglishDigits(code) } });
        window.location.href = "/upload";
      }
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="shell narrow">
      <a href="/" className="brand"><span className="brandMark">↑</span> FileTransfer</a>
      <form className="card" onSubmit={submit}>
        <h2>{step === "phone" ? "ورود با شماره موبایل" : "کد تأیید را وارد کنید"}</h2>
        <p>{step === "phone" ? "کد تأیید برای شما پیامک می‌شود." : `کد به ${phone} ارسال شد.`}</p>
        {step === "phone" ? (
          <input key="phone" className="field ltr" inputMode="tel" autoComplete="tel" placeholder="۰۹۱۲۱۲۳۴۵۶۷" value={phone} onChange={(e) => setPhone(e.target.value)} autoFocus required />
        ) : (
          <input key="code" className="field ltr code" inputMode="numeric" autoComplete="one-time-code" placeholder="------" value={code} onChange={(e) => setCode(e.target.value)} autoFocus required />
        )}
        {error && <div className="error" role="alert">{error}</div>}
        <button className="primary wide" disabled={busy}>{busy ? "کمی صبر کنید…" : step === "phone" ? "ارسال کد" : "ورود"}</button>
        {step === "code" && <button type="button" className="login" onClick={() => { setStep("phone"); setCode(""); setError(""); }}>تغییر شماره</button>}
      </form>
    </main>
  );
}
