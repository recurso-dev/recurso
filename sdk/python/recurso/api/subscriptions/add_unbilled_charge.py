from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.add_unbilled_charge_body import AddUnbilledChargeBody
from ...models.error import Error
from ...models.unbilled_charge import UnbilledCharge
from ...types import Response


def _get_kwargs(
    id: UUID,
    *,
    body: AddUnbilledChargeBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/v1/subscriptions/{id}/charges".format(
            id=quote(str(id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | UnbilledCharge | None:
    if response.status_code == 201:
        response_201 = UnbilledCharge.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | UnbilledCharge]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddUnbilledChargeBody,
) -> Response[Error | UnbilledCharge]:
    """Add an unbilled (ad-hoc) charge

     The charge is picked up on the subscription's next invoice.

    Args:
        id (UUID):
        body (AddUnbilledChargeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | UnbilledCharge]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddUnbilledChargeBody,
) -> Error | UnbilledCharge | None:
    """Add an unbilled (ad-hoc) charge

     The charge is picked up on the subscription's next invoice.

    Args:
        id (UUID):
        body (AddUnbilledChargeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | UnbilledCharge
    """

    return sync_detailed(
        id=id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddUnbilledChargeBody,
) -> Response[Error | UnbilledCharge]:
    """Add an unbilled (ad-hoc) charge

     The charge is picked up on the subscription's next invoice.

    Args:
        id (UUID):
        body (AddUnbilledChargeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | UnbilledCharge]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: AddUnbilledChargeBody,
) -> Error | UnbilledCharge | None:
    """Add an unbilled (ad-hoc) charge

     The charge is picked up on the subscription's next invoice.

    Args:
        id (UUID):
        body (AddUnbilledChargeBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | UnbilledCharge
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            body=body,
        )
    ).parsed
