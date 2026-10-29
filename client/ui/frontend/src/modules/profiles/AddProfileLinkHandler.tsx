import { useCallback, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Events, Window } from "@wailsio/runtime";
import { DeepLink, Settings as SettingsSvc } from "@bindings/services";
import { type AddProfileLink, SetConfigParams } from "@bindings/services/models.js";
import { useProfile } from "@/contexts/ProfileContext";
import { useConfirm } from "@/contexts/DialogContext";
import { isNetbirdCloud } from "@/hooks/useManagementUrl.ts";
import { errorDialog, formatErrorMessage } from "@/lib/errors";

// Sightwell fork: sightwell://add-profile links from a tenant's dashboard.
// The Go side (services/deeplink.go) has already parsed and validated the
// link; nothing changes here until the user confirms.
const EVENT_LINK_RECEIVED = "sightwell:link:received";
// Every window hears the event; only the main window, which the Go side
// brings forward, offers the link.
const MAIN_WINDOW = "main";

const sameManagementUrl = (a: string, b: string) => {
    const norm = (u: string) => u.trim().toLowerCase().replace(/\/+$/, "").replace(/:443$/, "");
    return norm(a) === norm(b);
};

export const AddProfileLinkHandler = () => {
    const { t } = useTranslation();
    const confirm = useConfirm();
    const { loaded, username, profiles, addProfile, switchProfile } = useProfile();

    // The event listener outlives renders, so it reads the latest profile
    // state through a ref.
    const state = useRef({ loaded, username, profiles, addProfile, switchProfile });
    state.current = { loaded, username, profiles, addProfile, switchProfile };
    const handling = useRef(false);
    const isMain = useRef<Promise<boolean> | null>(null);

    // A profile already made from the same link is reused, so a second click
    // selects it instead of adding a duplicate.
    const findOrAddProfile = useCallback(async (link: AddProfileLink) => {
        const { username, profiles, addProfile } = state.current;
        for (const p of profiles.filter((p) => p.name === link.name)) {
            const config = await SettingsSvc.GetConfig({ profileName: p.id, username });
            if (sameManagementUrl(config.managementUrl, link.managementUrl)) return p.id;
        }
        const id = await addProfile(link.name);
        if (!isNetbirdCloud(link.managementUrl)) {
            await SettingsSvc.SetConfig(
                new SetConfigParams({
                    profileName: id,
                    username,
                    managementUrl: link.managementUrl,
                }),
            );
        }
        return id;
    }, []);

    const handlePending = useCallback(async () => {
        if (handling.current || !state.current.loaded) return;
        isMain.current ??= Window.Name().then((name) => name === MAIN_WINDOW);
        if (!(await isMain.current)) return;
        if (handling.current) return;
        handling.current = true;
        try {
            const pending = await DeepLink.TakePending();
            if (pending.refusal) {
                await errorDialog({
                    Title: t("link.refused.title"),
                    Message: t(`link.refused.${pending.refusal}`),
                });
                return;
            }
            const link = pending.link;
            if (!link) return;

            const ok = await confirm({
                title: t("link.addProfile.title", { name: link.name }),
                description: t("link.addProfile.message", { host: link.host }),
                confirmLabel: t("link.addProfile.confirm"),
            });
            if (!ok) return;

            try {
                const id = await findOrAddProfile(link);
                // The connecting switch opens browser sign-in when the new
                // profile needs it.
                await state.current.switchProfile(id);
            } catch (e) {
                await errorDialog({
                    Title: t("link.addProfile.errorTitle"),
                    Message: formatErrorMessage(e),
                });
            }
        } finally {
            handling.current = false;
        }
    }, [t, confirm, findOrAddProfile]);

    useEffect(() => {
        const off = Events.On(EVENT_LINK_RECEIVED, () => {
            handlePending().catch((err: unknown) =>
                console.error("[AddProfileLinkHandler] link failed", err),
            );
        });
        return () => {
            off();
        };
    }, [handlePending]);

    // A link that arrived before the profiles loaded is picked up once they have.
    useEffect(() => {
        if (!loaded) return;
        handlePending().catch((err: unknown) =>
            console.error("[AddProfileLinkHandler] link failed", err),
        );
    }, [loaded, handlePending]);

    return null;
};
