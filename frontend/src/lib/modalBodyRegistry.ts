import AlertModalBody from "../components/modals/AlertModalBody.svelte";
import ConfirmModalBody from "../components/modals/ConfirmModalBody.svelte";
import PromptModalBody from "../components/modals/PromptModalBody.svelte";
import FolderPickerModalBody from "../components/modals/FolderPickerModalBody.svelte";
import CrashReportModalBody from "../components/modals/CrashReportModalBody.svelte";
import UpdateModalBody from "../components/modals/UpdateModalBody.svelte";

export const modalBodyComponents: Record<string, any> = {
  alert: AlertModalBody,
  alertNoButton: AlertModalBody,
  confirm: ConfirmModalBody,
  prompt: PromptModalBody,
  folder: FolderPickerModalBody,
  crashReport: CrashReportModalBody,
  update: UpdateModalBody,
};
