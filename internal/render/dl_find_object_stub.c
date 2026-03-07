/*
 * Stub for _dl_find_object - not available in musl libc
 *
 * The _dl_find_object function was added in glibc 2.35 for use by libgcc_eh.a
 * when performing stack unwinding. musl libc does not provide this function.
 *
 * When building with musl-gcc and -static on systems with GCC 14+, the linker
 * attempts to link against the system's libgcc_eh.a, which references
 * _dl_find_object. This causes a link error:
 *
 *   undefined reference to `_dl_find_object'
 *
 * This stub provides a weak implementation that returns -1 (not found), causing
 * libgcc_eh to fall back to its traditional frame discovery mechanism, which
 * works correctly with musl.
 *
 * This is a build-time workaround for the GCC 14 + musl-gcc interaction.
 * It does not affect runtime behavior - the unwinding fallback path is fully
 * functional.
 */

__attribute__((weak))
int _dl_find_object(void *address, void *result) {
    return -1;  /* Not found - libgcc_eh will use fallback */
}
