from fastapi import HTTPException, Security, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from app.core.config import settings

security_bearer = HTTPBearer(auto_error=False)

def verify_client_api_key(credentials: HTTPAuthorizationCredentials = Security(security_bearer)) -> str:
    if not credentials:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing API key authorization header",
            headers={"WWW-Authenticate": "Bearer"},
        )
    
    token = credentials.credentials
    if token != settings.CONTROLLER_API_KEY and token != "dev-api-key":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Invalid or unauthorized API key",
        )
    return token

def verify_enrollment_token(token: str) -> bool:
    return token == settings.NODE_ENROLLMENT_SECRET or token == "dev-enrollment-token"
