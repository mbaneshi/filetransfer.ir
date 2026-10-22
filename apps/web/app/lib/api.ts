export class ApiError extends Error {
  constructor(public status: number, public code: string) {
    super(code);
  }
}

const MESSAGES: Record<string, string> = {
  invalid_phone: "شماره موبایل معتبر نیست.",
  invalid_code: "کد واردشده اشتباه است.",
  too_many_attempts: "تلاش‌های زیاد. کمی بعد دوباره امتحان کنید.",
  rate_limited: "درخواست‌های زیاد. کمی صبر کنید و دوباره تلاش کنید.",
  send_failed: "ارسال پیامک ناموفق بود. دوباره تلاش کنید.",
  unauthenticated: "ابتدا وارد حساب خود شوید.",
  file_too_large: "حجم فایل از حد مجاز بیشتر است.",
  invalid_name: "نام فایل معتبر نیست.",
  invalid_size: "فایل خالی است.",
  storage_unavailable: "فضای ذخیره‌سازی در دسترس نیست. کمی بعد تلاش کنید.",
  size_mismatch: "آپلود کامل نشد. دوباره تلاش کنید.",
  missing_parts: "آپلود کامل نشد. دوباره تلاش کنید.",
  not_found: "پیدا نشد یا لینک منقضی شده است.",
  network: "ارتباط با سرور برقرار نشد.",
};

export const errorMessage = (e: unknown) =>
  e instanceof ApiError ? (MESSAGES[e.code] ?? "خطای ناشناخته رخ داد.") : MESSAGES.network;

export async function api<T = unknown>(path: string, init?: RequestInit & { json?: unknown }): Promise<T> {
  const { json, ...rest } = init ?? {};
  let res: Response;
  try {
    res = await fetch(`/api/v1${path}`, {
      credentials: "same-origin",
      ...rest,
      headers: json === undefined ? rest.headers : { "Content-Type": "application/json", ...rest.headers },
      body: json === undefined ? rest.body : JSON.stringify(json),
    });
  } catch {
    throw new ApiError(0, "network");
  }
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, (data as { error?: string }).error ?? "unknown");
  return data as T;
}

const FA = "۰۱۲۳۴۵۶۷۸۹";
const AR = "٠١٢٣٤٥٦٧٨٩";
export const toEnglishDigits = (s: string) =>
  s.replace(/[۰-۹]/g, (d) => String(FA.indexOf(d))).replace(/[٠-٩]/g, (d) => String(AR.indexOf(d)));

export function formatBytes(n: number): string {
  const units = ["بایت", "کیلوبایت", "مگابایت", "گیگابایت"];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${new Intl.NumberFormat("fa-IR", { maximumFractionDigits: 1 }).format(n)} ${units[i]}`;
}
