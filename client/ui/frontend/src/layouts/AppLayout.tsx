import { Outlet } from "react-router-dom";
import { ClientVersionProvider } from "@/contexts/ClientVersionContext.tsx";
import { StatusProvider } from "@/contexts/StatusContext.tsx";
import { DebugBundleProvider } from "@/contexts/DebugBundleContext.tsx";
import { ProfileProvider } from "@/contexts/ProfileContext.tsx";
import { DialogProvider } from "@/contexts/DialogContext.tsx";
import { RestrictionsProvider } from "@/contexts/RestrictionsContext.tsx";
import { ReadySignal } from "@/components/ReadySignal.tsx";
import { AddProfileLinkHandler } from "@/modules/profiles/AddProfileLinkHandler.tsx";

export const AppLayout = () => {
    return (
        <div className={"relative flex h-full flex-col"}>
            <DialogProvider>
                <StatusProvider>
                    <ProfileProvider>
                        <RestrictionsProvider>
                            <DebugBundleProvider>
                                <ClientVersionProvider>
                                    <Outlet />
                                    {/* Sightwell fork: before ReadySignal so it listens before queued events flush. */}
                                    <AddProfileLinkHandler />
                                    <ReadySignal />
                                </ClientVersionProvider>
                            </DebugBundleProvider>
                        </RestrictionsProvider>
                    </ProfileProvider>
                </StatusProvider>
            </DialogProvider>
        </div>
    );
};
