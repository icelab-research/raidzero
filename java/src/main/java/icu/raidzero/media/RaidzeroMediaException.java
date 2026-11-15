package icu.raidzero.media;

/** thrown by every client method on a non-2xx API response. */
public class RaidzeroMediaException extends Exception {

    private final String code;
    private final int status;

    /**
     * Constructs a new RaidzeroMediaException with the API error code, message, and HTTP status.
     *
     * @param code    machine-readable API error code
     * @param message human-readable error message used as the exception message
     * @param status  HTTP status code associated with the API response
     */
    public RaidzeroMediaException(String code, String message, int status) {
        super(message);
        this.code = code;
        this.status = status;
    }

    /**
 * The API's machine-readable error code.
 *
 * @return the machine-readable error code from the API (e.g., {@code "not_found"}).
 */
    public String getCode() { return code; }

    /**
 * HTTP status code associated with the API response.
 *
 * @return the stored HTTP status code for this exception
 */
    public int getStatus() { return status; }
}
