export type Lang = "en" | "vi";

export function getLang(args: string[]): Lang {
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--lang" && i + 1 < args.length) {
      return args[i + 1] === "vi" ? "vi" : "en";
    }
  }
  return "en";
}

export function withoutLang(args: string[]): string[] {
  const result: string[] = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--lang") {
      i++;
    } else {
      result.push(args[i]);
    }
  }
  return result;
}

export function resolveLang(
  localOpts: { lang?: string },
  parentOpts: { lang?: string }
): Lang {
  const raw =
    localOpts.lang && localOpts.lang !== "en"
      ? localOpts.lang
      : parentOpts.lang && parentOpts.lang !== "en"
        ? parentOpts.lang
        : "en";
  return raw === "vi" ? "vi" : "en";
}

export function startHereTitle(lang: Lang): string {
  return lang === "vi"
    ? "Bắt đầu — một số lệnh để bạn khởi động:"
    : "Start here — a few commands to get you going:";
}

export function categoryGettingStarted(lang: Lang): string {
  return lang === "vi" ? "Bắt đầu" : "Getting started";
}

export function categoryDailyTasks(lang: Lang): string {
  return lang === "vi" ? "Tác vụ hằng ngày" : "Daily tasks";
}

export function categoryHealthRecovery(lang: Lang): string {
  return lang === "vi" ? "Sức khỏe & khôi phục" : "Health & recovery";
}

export function categoryAutomation(lang: Lang): string {
  return lang === "vi" ? "Tự động hóa" : "Automation";
}

export function discoverMore(lang: Lang): string {
  return lang === "vi" ? "Khám phá thêm:" : "Discover more:";
}

export function newToXHarness(lang: Lang): string {
  return lang === "vi"
    ? "Mới sử dụng x-harness? Xem docs/GETTING_STARTED.md"
    : "New to x-harness? See docs/GETTING_STARTED.md";
}

export function usageLabel(lang: Lang): string {
  return lang === "vi" ? "Cách dùng:" : "Usage:";
}

export function forCommandSpecificHelp(lang: Lang): string {
  return lang === "vi"
    ? "Để xem trợ giúp cho từng lệnh:"
    : "For command-specific help:";
}

export function advancedLabel(lang: Lang): string {
  return lang === "vi" ? "Nâng cao:" : "Advanced:";
}

export function globalOptionsLabel(lang: Lang): string {
  return lang === "vi" ? "Tùy chọn toàn cục:" : "Global options:";
}

export function showHelpText(lang: Lang): string {
  return lang === "vi" ? "Hiển thị trợ giúp" : "Show help";
}

export function showAllCommandsText(lang: Lang): string {
  return lang === "vi" ? "Hiển thị tất cả lệnh" : "Show all commands";
}

export function showMaturityLabelsText(lang: Lang): string {
  return lang === "vi"
    ? "Hiển thị nhãn độ ổn định cho tất cả lệnh"
    : "Show help with maturity labels for all commands";
}

export function showVersionText(lang: Lang): string {
  return lang === "vi" ? "Hiển thị phiên bản" : "Show version";
}

export function beginnerActionsTitle(lang: Lang): string {
  return lang === "vi"
    ? "# xh Hành động dành cho người mới"
    : "# xh Beginner Actions";
}

export function invokeUsingEither(lang: Lang): string {
  return lang === "vi"
    ? "Gọi bằng một trong hai cách:"
    : "Invoke using either:";
}

export function installedCLIText(lang: Lang): string {
  return lang === "vi" ? "CLI đã cài:" : "Installed CLI:";
}

export function localSourceText(lang: Lang): string {
  return lang === "vi" ? "Mã nguồn cục bộ:" : "Local source:";
}

export function forMoreInfo(lang: Lang): string {
  return lang === "vi"
    ? "Để biết thêm: xh <command> --help"
    : "For more info: xh <command> --help";
}

export function discoverHelpDesc(lang: Lang): string {
  return lang === "vi"
    ? "Các lệnh thường dùng và cách sử dụng"
    : "Common commands and usage";
}

export function discoverHelpAllDesc(lang: Lang): string {
  return lang === "vi" ? "Tất cả lệnh" : "All commands";
}

export function discoverHelpMaturityDesc(lang: Lang): string {
  return lang === "vi"
    ? "Các lệnh theo nhóm độ ổn định"
    : "Commands grouped by stability";
}

export function actionHeader(lang: Lang): string {
  return lang === "vi" ? "Hành động" : "Action";
}

export function descriptionHeader(lang: Lang): string {
  return lang === "vi" ? "Mô tả" : "Description";
}

export function getBeginnerCommandDesc(name: string, lang: Lang): string {
  if (lang === "vi") {
    switch (name) {
      case "check":
        return "Chạy xác minh chỉ đọc đối với completion card";
      case "prepare":
        return "Kiểm tra xem workspace đã sẵn sàng cho việc bàn giao tác vụ tác nhân chưa";
      case "recover":
        return "Lấy gợi ý playbook khôi phục từ lỗi hoặc trace";
      case "status":
        return "Hiển thị tóm tắt trace";
      case "reset":
        return "Dọn dẹp trạng thái đã tạo của harness";
      case "start":
        return "Hướng dẫn bắt đầu: doctor, examples verify, init wizard, next steps";
      case "learn":
        return "Chuyến tham quan khái niệm chỉ đọc dành cho người mới";
      case "quick":
        return "Công cụ gợi ý hành động tiếp theo chỉ đọc dành cho người mới";
      case "run":
        return "Chạy một workflow recipe có sẵn";
      case "ci":
        return "Chạy CI workflow có sẵn";
      case "init":
        return "Cài đặt tài sản harness vào workspace";
      case "add":
        return "Thêm claim, evidence, hoặc completion card helpers";
      case "doctor":
        return "Xác thực sức khỏe và cấu hình workspace";
      case "actions":
        return "Liệt kê các hành động thân thiện với người mới";
    }
  }
  switch (name) {
    case "check":
      return "Run read-only verification against a completion card";
    case "prepare":
      return "Check if workspace is ready for agent task handoff";
    case "recover":
      return "Get recovery playbook suggestions from errors or trace";
    case "status":
      return "Show trace summary";
    case "reset":
      return "Clean generated harness state";
    case "start":
      return "Guided onboarding: doctor, examples verify, init wizard, next steps";
    case "learn":
      return "Read-only concept tour for beginners";
    case "quick":
      return "Read-only next-action recommender for newcomers";
    case "run":
      return "Run a built-in workflow recipe";
    case "ci":
      return "Run the built-in CI workflow";
    case "init":
      return "Install harness assets into a workspace";
    case "add":
      return "Add claim, evidence, or completion card helpers";
    case "doctor":
      return "Validate workspace health and configuration";
    case "actions":
      return "List beginner-friendly actions";
  }
  return "";
}

export function learnTitle(lang: Lang): string {
  return lang === "vi"
    ? "xh learn - Khái niệm cơ bản"
    : "xh learn - Concept tour";
}

export function learnOverview(lang: Lang): string {
  return lang === "vi" ? "Tổng quan" : "Overview";
}

export function learnCoreConcepts(lang: Lang): string {
  return lang === "vi" ? "Khái niệm cốt lõi" : "Core concepts";
}

export function learnTiersAndEvidence(lang: Lang): string {
  return lang === "vi" ? "Cấp độ và bằng chứng" : "Tiers and evidence";
}

export function learnNextStepsLabel(lang: Lang): string {
  return lang === "vi" ? "Bước tiếp theo:" : "Next steps:";
}

export function learnOverviewBody(lang: Lang): string {
  return lang === "vi"
    ? "x-harness là một bộ kiểm tra nhẹ dành cho quy trình làm việc của tác nhân AI. Nó thực thi việc hoàn thành được phép, không được tự nhận, thông qua trình xác minh chỉ đọc."
    : "x-harness is a lightweight verify-gated harness for AI-agent workflows. It enforces that completion is admitted, not claimed, via a read-only verifier.";
}

export function learnCoreConceptsBody(lang: Lang): string {
  return lang === "vi"
    ? `Hoàn thành được phép, không được tự nhận — chỉ cổng xác minh mới có thể chấp nhận công việc.
Trình xác minh chỉ đọc — nó kiểm tra bằng chứng nhưng không bao giờ chỉnh sửa tệp nguồn.
Thành công là kết quả duy nhất được chấp nhận — mọi kết quả không thành công đều bị giữ lại.
Các cấp độ chuẩn là light, standard và deep — mỗi cấp có yêu cầu bằng chứng ngày càng tăng.
PGV (xác minh trước cổng) chỉ mang tính tư vấn — nó không bao giờ ghi đè cổng xác minh.`
    : `Completion is admitted, not claimed — only the verify gate can accept work.
Verifier is read-only — it inspects evidence but never edits source files.
Success is the only accepted outcome — all non-success results are withheld.
Canonical tiers are light, standard, and deep — each with increasing evidence requirements.
PGV (pre-gate validation) is advisory-only — it never overrides the verify gate.`;
}

export function learnTiersAndEvidenceBody(lang: Lang): string {
  return lang === "vi"
    ? `light: files_changed cộng với command evidence hoặc manual rationale.
standard: thêm done_checklist và prediction.
deep: thêm evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets và verification artifacts.`
    : `light: files_changed plus command evidence or manual rationale.
standard: adds done_checklist and prediction.
deep: adds evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets, and verification artifacts.`;
}

export function learnNextSteps(lang: Lang): string[] {
  return lang === "vi"
    ? [
        "Chạy xh start để bắt đầu hướng dẫn",
        "Chạy xh check --card <card> để xác minh completion card",
        "Chạy xh actions để xem các lệnh thân thiện với người mới",
        "Đọc docs/GETTING_STARTED.md",
      ]
    : [
        "Run xh start for guided onboarding",
        "Run xh check --card <card> to verify a completion card",
        "Run xh actions to see beginner-friendly commands",
        "Read docs/GETTING_STARTED.md",
      ];
}

export function quickTitle(lang: Lang): string {
  return lang === "vi"
    ? "xh quick - Gợi ý hành động tiếp theo"
    : "xh quick - Next-action recommender";
}

export function quickRootLabel(lang: Lang): string {
  return lang === "vi" ? "thư mục gốc" : "root";
}

export function quickRecommendationLabel(lang: Lang): string {
  return lang === "vi" ? "gợi ý" : "recommendation";
}

export function quickReasonLabel(lang: Lang): string {
  return lang === "vi" ? "lý do" : "reason";
}

export function quickDetectedSignalsLabel(lang: Lang): string {
  return lang === "vi" ? "Tín hiệu phát hiện:" : "Detected signals:";
}

export function quickNoneLabel(lang: Lang): string {
  return lang === "vi" ? "  (không có)" : "  (none)";
}

export function quickNextStepsLabel(lang: Lang): string {
  return lang === "vi" ? "Bước tiếp theo:" : "Next steps:";
}

export function startTitle(lang: Lang): string {
  return lang === "vi"
    ? "xh start - Hướng dẫn bắt đầu"
    : "xh start - Guided onboarding";
}

export function startStepLabel(name: string, lang: Lang): string {
  if (lang === "vi") {
    switch (name) {
      case "doctor":
        return "doctor";
      case "examples_verify":
        return "xác minh examples";
      case "init_wizard":
        return "init wizard";
    }
  }
  switch (name) {
    case "examples_verify":
      return "examples verify";
    case "init_wizard":
      return "init wizard";
  }
  return name;
}

export function startNextStepsTitle(lang: Lang): string {
  return lang === "vi" ? "Bước tiếp theo:" : "Next steps:";
}

export function startFirstVerification(lang: Lang): string {
  return lang === "vi"
    ? "Chạy xác minh đầu tiên: xh check --card completion-card.yaml"
    : "Run your first verification: xh check --card completion-card.yaml";
}

export function startReadDocs(lang: Lang): string {
  return lang === "vi"
    ? "Đọc tài liệu: docs/GETTING_STARTED.md"
    : "Read the docs: docs/GETTING_STARTED.md";
}
