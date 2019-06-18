package security

import (
	"github.com/dgrijalva/jwt-go"
)

type JWTManager interface {
	Generate(spiffeID string, networkService string, obo *jwt.Token) (string, error)
	Verify(tokeString string) error
}

type nsmClaims struct {
	jwt.StandardClaims
	Obo string `json:"obo"`
}

//type jwtManager struct {
//	mgr Manager
//}

//func NewJWTManager(crtMgr Manager) JWTManager {
//	return &jwtManager{
//		mgr: crtMgr,
//	}
//}

//func (m *jwtManager) Verify(tokenString string) error {
//	token, parts, err := new(jwt.Parser).ParseUnverified(tokenString, &nsmClaims{})
//	if err != nil {
//		return err
//	}
//
//	claims, ok := token.Claims.(*nsmClaims);
//	if !ok {
//		return errors.New("wrong claims format")
//	}
//
//	cert := m.mgr.GetCertificateBySpiffeID(claims.Subject)
//	if cert == nil {
//		return fmt.Errorf("no certificate proveded for %s", claims.Subject)
//	}
//
//	_, err = cert.Verify(x509.VerifyOptions{
//		Roots: m.mgr.GetCABundle(),
//	})
//
//	if err != nil {
//		return fmt.Errorf("certificate is signed by untrusted authority: %s", err.Error())
//	}
//
//	if err := token.Method.Verify(strings.Join(parts[0:2], "."), parts[2], cert.PublicKey); err != nil {
//		return fmt.Errorf("jwt signature is not valid: %s", err.Error())
//	}
//
//	return nil
//}
//
//func (m *jwtManager) Generate(spiffeID string, networkService string, obo *jwt.Token) (string, error) {
//	var oboClaim string
//	if obo != nil {
//		oboClaim = obo.Raw
//	}
//
//	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &nsmClaims{
//		StandardClaims: jwt.StandardClaims{
//			Audience: networkService,
//			Issuer:   "test",
//			Subject:  spiffeID,
//		},
//		Obo: oboClaim,
//	})
//
//	return token.SignedString(m.mgr.GetCertificate().PrivateKey)
//}
