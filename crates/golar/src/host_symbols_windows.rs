use crate::bindings::{FileWithProgram, RawProgram, Workspace};
use crate::common::{RawNode, RawType};

use std::ffi::c_char;
use std::sync::OnceLock;

macro_rules! define_windows_wrappers {
    ($(fn $name:ident($($arg:ident: $arg_ty:ty),* $(,)?) $(-> $ret:ty)?;)+) => {
        $(
            #[allow(clippy::too_many_arguments)]
            pub(crate) fn $name($($arg: $arg_ty),*) $(-> $ret)? {
                unsafe { (bindings().$name)($($arg),*) }
            }
        )+
    };
}

macro_rules! define_windows_bindings_struct {
    ($(fn $name:ident($($arg:ident: $arg_ty:ty),* $(,)?) $(-> $ret:ty)?;)+) => {
        struct GolarBindings {
            _library: libloading::Library,
            $(
                $name: unsafe extern "C" fn($($arg_ty),*) $(-> $ret)?,
            )+
        }
    };
}

golar_host_symbols!(define_windows_bindings_struct);

static GOLAR_BINDINGS: OnceLock<GolarBindings> = OnceLock::new();

fn bindings() -> &'static GolarBindings {
    GOLAR_BINDINGS.get().expect("host symbols not initialized")
}

fn load_symbol<T: Copy>(library: &libloading::Library, name: &str) -> napi::Result<T> {
    let symbol = unsafe { library.get::<T>(name) }
        .map_err(|err| napi::Error::from_reason(format!("failed to load symbol {name}: {err}")))?;
    Ok(*symbol)
}

macro_rules! define_windows_load_bindings {
    ($(fn $name:ident($($arg:ident: $arg_ty:ty),* $(,)?) $(-> $ret:ty)?;)+) => {
        fn load_bindings(library: libloading::Library) -> napi::Result<GolarBindings> {
            Ok(GolarBindings {
                $(
                    $name: load_symbol(&library, stringify!($name))?,
                )+
                _library: library,
            })
        }
    };
}

golar_host_symbols!(define_windows_load_bindings);

pub(crate) fn setup(golar_addon_path: &str) -> napi::Result<()> {
    if GOLAR_BINDINGS.get().is_some() {
        return Ok(());
    }

    let library: libloading::Library =
        libloading::os::windows::Library::open_already_loaded(golar_addon_path)
            .map_err(|err| {
                napi::Error::from_reason(format!(
                    "failed to access already loaded golar addon at {golar_addon_path}: {err}"
                ))
            })?
            .into();

    let bindings = load_bindings(library)?;

    let _ = GOLAR_BINDINGS.set(bindings);
    Ok(())
}

pub(crate) fn verify_ready() -> napi::Result<()> {
    if GOLAR_BINDINGS.get().is_some() {
        Ok(())
    } else {
        Err(napi::Error::from_reason(
            "native addon setup(golarAddonPath) is expected to have been called",
        ))
    }
}

golar_host_symbols!(define_windows_wrappers);
