from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.portal_start_mandate_reauth_body import PortalStartMandateReauthBody
from ...models.portal_start_mandate_reauth_response_200 import PortalStartMandateReauthResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: PortalStartMandateReauthBody | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/portal/api/payment-method/mandate",
    }

    if not isinstance(body, Unset):
        _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | PortalStartMandateReauthResponse200 | None:
    if response.status_code == 200:
        response_200 = PortalStartMandateReauthResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 409:
        response_409 = Error.from_dict(response.json())

        return response_409

    if response.status_code == 422:
        response_422 = Error.from_dict(response.json())

        return response_422

    if response.status_code == 503:
        response_503 = Error.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | PortalStartMandateReauthResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalStartMandateReauthBody | Unset = UNSET,
) -> Response[Error | PortalStartMandateReauthResponse200]:
    """Re-authorize a UPI Autopay mandate

     Creates a fresh UPI mandate for the authenticated portal customer and returns Razorpay's hosted
    authorization URL. The cap is sized from the customer's billing history (2× the largest recent non-
    void invoice); activation happens only when Razorpay confirms the token via the `token.confirmed`
    webhook. Requires a phone number on the customer (Razorpay rejects recurring registration links
    without one) and real Razorpay keys on the deployment.

    Args:
        body (PortalStartMandateReauthBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalStartMandateReauthResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    body: PortalStartMandateReauthBody | Unset = UNSET,
) -> Error | PortalStartMandateReauthResponse200 | None:
    """Re-authorize a UPI Autopay mandate

     Creates a fresh UPI mandate for the authenticated portal customer and returns Razorpay's hosted
    authorization URL. The cap is sized from the customer's billing history (2× the largest recent non-
    void invoice); activation happens only when Razorpay confirms the token via the `token.confirmed`
    webhook. Requires a phone number on the customer (Razorpay rejects recurring registration links
    without one) and real Razorpay keys on the deployment.

    Args:
        body (PortalStartMandateReauthBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalStartMandateReauthResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalStartMandateReauthBody | Unset = UNSET,
) -> Response[Error | PortalStartMandateReauthResponse200]:
    """Re-authorize a UPI Autopay mandate

     Creates a fresh UPI mandate for the authenticated portal customer and returns Razorpay's hosted
    authorization URL. The cap is sized from the customer's billing history (2× the largest recent non-
    void invoice); activation happens only when Razorpay confirms the token via the `token.confirmed`
    webhook. Requires a phone number on the customer (Razorpay rejects recurring registration links
    without one) and real Razorpay keys on the deployment.

    Args:
        body (PortalStartMandateReauthBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalStartMandateReauthResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PortalStartMandateReauthBody | Unset = UNSET,
) -> Error | PortalStartMandateReauthResponse200 | None:
    """Re-authorize a UPI Autopay mandate

     Creates a fresh UPI mandate for the authenticated portal customer and returns Razorpay's hosted
    authorization URL. The cap is sized from the customer's billing history (2× the largest recent non-
    void invoice); activation happens only when Razorpay confirms the token via the `token.confirmed`
    webhook. Requires a phone number on the customer (Razorpay rejects recurring registration links
    without one) and real Razorpay keys on the deployment.

    Args:
        body (PortalStartMandateReauthBody | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalStartMandateReauthResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
