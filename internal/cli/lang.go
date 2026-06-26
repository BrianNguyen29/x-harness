package cli

type Lang string

const (
	LangEN Lang = "en"
	LangVI Lang = "vi"
)

func parseLang(args []string) (Lang, []string) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--lang" && i+1 < len(args) {
			lang := LangEN
			if args[i+1] == "vi" {
				lang = LangVI
			}
			clean := make([]string, 0, len(args)-2)
			clean = append(clean, args[:i]...)
			clean = append(clean, args[i+2:]...)
			return lang, clean
		}
	}
	return LangEN, args
}

func startHereTitle(l Lang) string {
	if l == LangVI {
		return "Bắt đầu — một số lệnh để bạn khởi động:"
	}
	return "Start here — a few commands to get you going:"
}

func categoryGettingStarted(l Lang) string {
	if l == LangVI {
		return "Bắt đầu"
	}
	return "Getting started"
}

func categoryDailyTasks(l Lang) string {
	if l == LangVI {
		return "Tác vụ hằng ngày"
	}
	return "Daily tasks"
}

func categoryHealthRecovery(l Lang) string {
	if l == LangVI {
		return "Sức khỏe & khôi phục"
	}
	return "Health & recovery"
}

func categoryAutomation(l Lang) string {
	if l == LangVI {
		return "Tự động hóa"
	}
	return "Automation"
}

func discoverMore(l Lang) string {
	if l == LangVI {
		return "Khám phá thêm:"
	}
	return "Discover more:"
}

func newToXHarness(l Lang) string {
	if l == LangVI {
		return "Mới sử dụng x-harness? Xem docs/GETTING_STARTED.md"
	}
	return "New to x-harness? See docs/GETTING_STARTED.md"
}

func usageLabel(l Lang) string {
	if l == LangVI {
		return "Cách dùng:"
	}
	return "Usage:"
}

func forCommandSpecificHelp(l Lang) string {
	if l == LangVI {
		return "Để xem trợ giúp cho từng lệnh:"
	}
	return "For command-specific help:"
}

func advancedLabel(l Lang) string {
	if l == LangVI {
		return "Nâng cao:"
	}
	return "Advanced:"
}

func globalOptionsLabel(l Lang) string {
	if l == LangVI {
		return "Tùy chọn toàn cục:"
	}
	return "Global options:"
}

func showHelpText(l Lang) string {
	if l == LangVI {
		return "Hiển thị trợ giúp"
	}
	return "Show help"
}

func showAllCommandsText(l Lang) string {
	if l == LangVI {
		return "Hiển thị tất cả lệnh"
	}
	return "Show all commands"
}

func showMaturityLabelsText(l Lang) string {
	if l == LangVI {
		return "Hiển thị nhãn độ ổn định cho tất cả lệnh"
	}
	return "Show help with maturity labels for all commands"
}

func showVersionText(l Lang) string {
	if l == LangVI {
		return "Hiển thị phiên bản"
	}
	return "Show version"
}

func beginnerActionsTitle(l Lang) string {
	if l == LangVI {
		return "# xh Hành động dành cho người mới"
	}
	return "# xh Beginner Actions"
}

func invokeUsingEither(l Lang) string {
	if l == LangVI {
		return "Gọi bằng một trong hai cách:"
	}
	return "Invoke using either:"
}

func installedCLIText(l Lang) string {
	if l == LangVI {
		return "CLI đã cài:"
	}
	return "Installed CLI:"
}

func localSourceText(l Lang) string {
	if l == LangVI {
		return "Mã nguồn cục bộ:"
	}
	return "Local source:"
}

func actionHeader(l Lang) string {
	if l == LangVI {
		return "Hành động"
	}
	return "Action"
}

func descriptionHeader(l Lang) string {
	if l == LangVI {
		return "Mô tả"
	}
	return "Description"
}

func forMoreInfoText(l Lang) string {
	if l == LangVI {
		return "Để biết thêm: xh <command> --help"
	}
	return "For more info: xh <command> --help"
}

func discoverHelpDesc(l Lang) string {
	if l == LangVI {
		return "Các lệnh thường dùng và cách sử dụng"
	}
	return "Common commands and usage"
}

func discoverHelpAllDesc(l Lang) string {
	if l == LangVI {
		return "Tất cả lệnh"
	}
	return "All commands"
}

func discoverHelpMaturityDesc(l Lang) string {
	if l == LangVI {
		return "Các lệnh theo nhóm độ ổn định"
	}
	return "Commands grouped by stability"
}

func beginnerCommandDesc(name string, lang Lang) string {
	if lang == LangVI {
		switch name {
		case "verify":
			return "Chạy cổng xác minh chỉ đọc đối với completion card"
		case "check":
			return "Chạy xác minh chỉ đọc đối với completion card"
		case "prepare":
			return "Kiểm tra xem workspace đã sẵn sàng cho việc bàn giao tác vụ tác nhân chưa"
		case "recover":
			return "Lấy gợi ý playbook khôi phục từ lỗi hoặc trace"
		case "status":
			return "Hiển thị tóm tắt trace"
		case "reset":
			return "Dọn dẹp trạng thái đã tạo của harness"
		case "start":
			return "Hướng dẫn bắt đầu: doctor, examples verify, init wizard, next steps"
		case "learn":
			return "Chuyến tham quan khái niệm chỉ đọc dành cho người mới"
		case "quick":
			return "Công cụ gợi ý hành động tiếp theo chỉ đọc dành cho người mới"
		case "run":
			return "Chạy một workflow recipe có sẵn"
		case "ci":
			return "Chạy CI workflow có sẵn"
		case "init":
			return "Cài đặt tài sản harness vào workspace"
		case "add":
			return "Thêm claim, evidence, hoặc completion card helpers"
		case "doctor":
			return "Xác thực sức khỏe và cấu hình workspace"
		case "actions":
			return "Liệt kê các hành động thân thiện với người mới"
		}
	}
	switch name {
	case "check":
		return "Run read-only verification against a completion card"
	case "prepare":
		return "Check if workspace is ready for agent task handoff"
	case "recover":
		return "Get recovery playbook suggestions from errors or trace"
	case "status":
		return "Show trace summary"
	case "reset":
		return "Clean generated harness state"
	}
	for _, c := range commands {
		if c.Name == name {
			return c.Description
		}
	}
	return ""
}

func learnTitle(l Lang) string {
	if l == LangVI {
		return "xh learn - Khái niệm cơ bản"
	}
	return "xh learn - Concept tour"
}

func learnOverview(l Lang) string {
	if l == LangVI {
		return "Tổng quan"
	}
	return "Overview"
}

func learnCoreConcepts(l Lang) string {
	if l == LangVI {
		return "Khái niệm cốt lõi"
	}
	return "Core concepts"
}

func learnTiersAndEvidence(l Lang) string {
	if l == LangVI {
		return "Cấp độ và bằng chứng"
	}
	return "Tiers and evidence"
}

func learnNextStepsLabel(l Lang) string {
	if l == LangVI {
		return "Bước tiếp theo:"
	}
	return "Next steps:"
}

func learnOverviewBody(l Lang) string {
	if l == LangVI {
		return "x-harness là một bộ kiểm tra nhẹ dành cho quy trình làm việc của tác nhân AI. Nó thực thi việc hoàn thành được phép, không được tự nhận, thông qua trình xác minh chỉ đọc."
	}
	return "x-harness is a lightweight verify-gated harness for AI-agent workflows. It enforces that completion is admitted, not claimed, via a read-only verifier."
}

func learnCoreConceptsBody(l Lang) string {
	if l == LangVI {
		return "Hoàn thành được phép, không được tự nhận — chỉ cổng xác minh mới có thể chấp nhận công việc.\nTrình xác minh chỉ đọc — nó kiểm tra bằng chứng nhưng không bao giờ chỉnh sửa tệp nguồn.\nThành công là kết quả duy nhất được chấp nhận — mọi kết quả không thành công đều bị giữ lại.\nCác cấp độ chuẩn là light, standard và deep — mỗi cấp có yêu cầu bằng chứng ngày càng tăng.\nPGV (xác minh trước cổng) chỉ mang tính tư vấn — nó không bao giờ ghi đè cổng xác minh."
	}
	return "Completion is admitted, not claimed — only the verify gate can accept work.\nVerifier is read-only — it inspects evidence but never edits source files.\nSuccess is the only accepted outcome — all non-success results are withheld.\nCanonical tiers are light, standard, and deep — each with increasing evidence requirements.\nPGV (pre-gate validation) is advisory-only — it never overrides the verify gate."
}

func learnTiersAndEvidenceBody(l Lang) string {
	if l == LangVI {
		return "light: files_changed cộng với command evidence hoặc manual rationale.\nstandard: thêm done_checklist và prediction.\ndeep: thêm evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets và verification artifacts."
	}
	return "light: files_changed plus command evidence or manual rationale.\nstandard: adds done_checklist and prediction.\ndeep: adds evidence scope declaration, untested regions, remaining risks, execution controls, rollback policy, read/write sets, and verification artifacts."
}

func learnNextSteps(l Lang) []string {
	if l == LangVI {
		return []string{
			"Chạy xh start để bắt đầu hướng dẫn",
			"Chạy xh check --card <card> để xác minh completion card",
			"Chạy xh actions để xem các lệnh thân thiện với người mới",
			"Đọc docs/GETTING_STARTED.md",
		}
	}
	return []string{
		"Run xh start for guided onboarding",
		"Run xh check --card <card> to verify a completion card",
		"Run xh actions to see beginner-friendly commands",
		"Read docs/GETTING_STARTED.md",
	}
}

func quickTitle(l Lang) string {
	if l == LangVI {
		return "xh quick - Gợi ý hành động tiếp theo"
	}
	return "xh quick - Next-action recommender"
}

func quickRootLabel(l Lang) string {
	if l == LangVI {
		return "thư mục gốc"
	}
	return "root"
}

func quickRecommendationLabel(l Lang) string {
	if l == LangVI {
		return "gợi ý"
	}
	return "recommendation"
}

func quickReasonLabel(l Lang) string {
	if l == LangVI {
		return "lý do"
	}
	return "reason"
}

func quickDetectedSignalsLabel(l Lang) string {
	if l == LangVI {
		return "Tín hiệu phát hiện:"
	}
	return "Detected signals:"
}

func quickNoneLabel(l Lang) string {
	if l == LangVI {
		return "  (không có)"
	}
	return "  (none)"
}

func quickNextStepsLabel(l Lang) string {
	if l == LangVI {
		return "Bước tiếp theo:"
	}
	return "Next steps:"
}

func startTitle(l Lang) string {
	if l == LangVI {
		return "xh start - Hướng dẫn bắt đầu"
	}
	return "xh start - Guided onboarding"
}

func startStepLabel(name string, l Lang) string {
	if l == LangVI {
		switch name {
		case "doctor":
			return "doctor"
		case "examples_verify":
			return "xác minh examples"
		case "init_wizard":
			return "init wizard"
		}
	}
	switch name {
	case "examples_verify":
		return "examples verify"
	case "init_wizard":
		return "init wizard"
	}
	return name
}

func startNextStepsTitle(l Lang) string {
	if l == LangVI {
		return "Bước tiếp theo:"
	}
	return "Next steps:"
}

func startFirstVerification(l Lang) string {
	if l == LangVI {
		return "Chạy xác minh đầu tiên: xh check --card completion-card.yaml"
	}
	return "Run your first verification: xh check --card completion-card.yaml"
}

func startReadDocs(l Lang) string {
	if l == LangVI {
		return "Đọc tài liệu: docs/GETTING_STARTED.md"
	}
	return "Read the docs: docs/GETTING_STARTED.md"
}
